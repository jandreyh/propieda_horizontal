// Package usecases orquesta la logica de aplicacion del modulo
// access_control. Cada usecase recibe sus dependencias por inyeccion
// (interfaces) y NO conoce HTTP ni la base.
package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	"github.com/saas-ph/api/internal/modules/access_control/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// QRGenerator genera un codigo QR plano nuevo. Inyectable para tests.
type QRGenerator func() (string, error)

// defaultQRGenerator produce 32 bytes aleatorios codificados en
// base64-urlsafe-no-padding. La capa de aplicacion entrega el plano UNA
// SOLA VEZ y solo persiste el sha256.
func defaultQRGenerator() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// HashQRCode aplica sha256 al codigo plano y devuelve el hash hex
// (lowercase). Es deterministico y reusable por checkin-by-qr.
func HashQRCode(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// CreatePreRegistration crea un pre-registro con QR firmado.
//
// Reglas:
//   - UnitID UUID valido.
//   - ExpiresAt > now() (no se permite crear ya expirado).
//   - MaxUses default = 1, debe ser >= 1.
//   - Si VisitorDocumentType viene, debe ser valido; si VisitorDocumentNumber
//     viene, no puede ser vacio (ya validado por longitud).
type CreatePreRegistration struct {
	Repo domain.PreRegistrationRepository
	// Now permite inyectar reloj para tests; si nil, time.Now.
	Now func() time.Time
	// QRGen permite inyectar generador de QR para tests; si nil, default.
	QRGen QRGenerator
}

// CreatePreRegistrationInput es el input del usecase (sin tags JSON).
type CreatePreRegistrationInput struct {
	UnitID                string
	CreatedByUserID       string
	VisitorFullName       string
	VisitorDocumentType   *string
	VisitorDocumentNumber *string
	ExpectedAt            *time.Time
	ExpiresAt             time.Time
	MaxUses               *int32
}

// CreatePreRegistrationOutput es el output del usecase. Incluye el QR
// plano (UNICA VEZ).
type CreatePreRegistrationOutput struct {
	Entity entities.PreRegistration
	QRCode string
}

// Execute valida y delega al repo. Devuelve el QR plano UNA SOLA VEZ.
func (u CreatePreRegistration) Execute(ctx context.Context, in CreatePreRegistrationInput) (CreatePreRegistrationOutput, error) {
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return CreatePreRegistrationOutput{}, apperrors.BadRequest("unit_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.CreatedByUserID); err != nil {
		return CreatePreRegistrationOutput{}, apperrors.BadRequest("created_by_user_id: " + err.Error())
	}
	if strings.TrimSpace(in.VisitorFullName) == "" {
		return CreatePreRegistrationOutput{}, apperrors.BadRequest("visitor_full_name is required")
	}
	now := u.now()
	if in.ExpiresAt.IsZero() || !in.ExpiresAt.After(now) {
		return CreatePreRegistrationOutput{}, apperrors.BadRequest("expires_at must be in the future")
	}
	maxUses, err := policies.ValidateMaxUses(in.MaxUses)
	if err != nil {
		return CreatePreRegistrationOutput{}, apperrors.BadRequest(err.Error())
	}
	if in.VisitorDocumentType != nil {
		if err := policies.ValidateDocumentType(*in.VisitorDocumentType); err != nil {
			return CreatePreRegistrationOutput{}, apperrors.BadRequest("visitor_document_type: " + err.Error())
		}
	}
	if in.VisitorDocumentNumber != nil {
		if err := policies.ValidateDocumentNumber(*in.VisitorDocumentNumber); err != nil {
			return CreatePreRegistrationOutput{}, apperrors.BadRequest("visitor_document_number: " + err.Error())
		}
	}

	gen := u.QRGen
	if gen == nil {
		gen = defaultQRGenerator
	}
	qrPlain, err := gen()
	if err != nil {
		return CreatePreRegistrationOutput{}, apperrors.Internal("failed to generate QR")
	}
	qrHash := HashQRCode(qrPlain)

	pre, err := u.Repo.Create(ctx, domain.CreatePreRegistrationInput{
		UnitID:                in.UnitID,
		CreatedByUserID:       in.CreatedByUserID,
		VisitorFullName:       strings.TrimSpace(in.VisitorFullName),
		VisitorDocumentType:   in.VisitorDocumentType,
		VisitorDocumentNumber: in.VisitorDocumentNumber,
		ExpectedAt:            in.ExpectedAt,
		ExpiresAt:             in.ExpiresAt,
		MaxUses:               maxUses,
		QRCodeHash:            qrHash,
	})
	if err != nil {
		return CreatePreRegistrationOutput{}, apperrors.Internal("failed to create pre-registration")
	}

	return CreatePreRegistrationOutput{Entity: pre, QRCode: qrPlain}, nil
}

func (u CreatePreRegistration) now() time.Time {
	if u.Now != nil {
		return u.Now()
	}
	return time.Now()
}

// mapPreregErr convierte sentinels del dominio en Problem HTTP.
func mapPreregErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrPreregistrationNotFound):
		return apperrors.NotFound("pre-registration not found")
	case errors.Is(err, domain.ErrPreregistrationExhausted):
		return apperrors.New(410, "preregistration-exhausted",
			"Pre-registration Exhausted",
			"the QR code has been fully consumed")
	case errors.Is(err, domain.ErrPreregistrationExpired):
		return apperrors.New(410, "preregistration-expired",
			"Pre-registration Expired",
			"the QR code has expired")
	}
	return apperrors.Internal("failed to consume pre-registration")
}
