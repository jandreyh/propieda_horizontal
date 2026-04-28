package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	"github.com/saas-ph/api/internal/modules/access_control/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// CheckinByQR redime un QR de pre-registro y registra la entrada del
// visitante.
//
// Flujo:
//  1. Hashea el QR plano y llama ConsumeOne (atomico). Si afecta 0 filas:
//     410 Gone (expirado/agotado/revocado).
//  2. Si el visitante tiene documento, chequea blacklist. Si hay match:
//     registra entrada con status='rejected', notes='blacklist hit: ...',
//     y devuelve 403.
//  3. Caso feliz: crea VisitorEntry con source='qr'.
type CheckinByQR struct {
	PreRegRepo    domain.PreRegistrationRepository
	BlacklistRepo domain.BlacklistRepository
	EntryRepo     domain.VisitorEntryRepository
}

// CheckinByQRInput es el input del usecase.
type CheckinByQRInput struct {
	QRCode   string
	GuardID  string
	PhotoURL *string
	Notes    *string
}

// Execute orquesta el flujo descrito en el doc del struct.
func (u CheckinByQR) Execute(ctx context.Context, in CheckinByQRInput) (entities.VisitorEntry, error) {
	if strings.TrimSpace(in.QRCode) == "" {
		return entities.VisitorEntry{}, apperrors.BadRequest("qr_code is required")
	}
	if err := policies.ValidateUUID(in.GuardID); err != nil {
		return entities.VisitorEntry{}, apperrors.BadRequest("guard_id: " + err.Error())
	}

	qrHash := HashQRCode(in.QRCode)
	pre, err := u.PreRegRepo.ConsumeOne(ctx, qrHash)
	if err != nil {
		return entities.VisitorEntry{}, mapPreregErr(err)
	}

	// Chequeo de blacklist si hay documento del visitante.
	if pre.VisitorDocumentType != nil && pre.VisitorDocumentNumber != nil &&
		*pre.VisitorDocumentType != "" && *pre.VisitorDocumentNumber != "" {
		bl, err := u.BlacklistRepo.Get(ctx,
			entities.DocumentType(*pre.VisitorDocumentType),
			*pre.VisitorDocumentNumber)
		if err != nil {
			return entities.VisitorEntry{}, apperrors.Internal("failed to check blacklist")
		}
		if bl != nil {
			// Registra intento por auditoria y rechaza.
			notes := fmt.Sprintf("blacklist hit: %s", bl.Reason)
			unitID := pre.UnitID
			preID := pre.ID
			_, _ = u.EntryRepo.Create(ctx, domain.CreateVisitorEntryInput{
				UnitID:                &unitID,
				PreRegistrationID:     &preID,
				VisitorFullName:       pre.VisitorFullName,
				VisitorDocumentType:   pre.VisitorDocumentType,
				VisitorDocumentNumber: *pre.VisitorDocumentNumber,
				PhotoURL:              in.PhotoURL,
				GuardID:               in.GuardID,
				Source:                entities.VisitorEntrySourceQR,
				Notes:                 &notes,
				Status:                entities.VisitorEntryStatusRejected,
			})
			return entities.VisitorEntry{}, mapBlacklistErr(bl.Reason)
		}
	}

	docNumber := ""
	if pre.VisitorDocumentNumber != nil {
		docNumber = *pre.VisitorDocumentNumber
	}
	unitID := pre.UnitID
	preID := pre.ID
	entry, err := u.EntryRepo.Create(ctx, domain.CreateVisitorEntryInput{
		UnitID:                &unitID,
		PreRegistrationID:     &preID,
		VisitorFullName:       pre.VisitorFullName,
		VisitorDocumentType:   pre.VisitorDocumentType,
		VisitorDocumentNumber: docNumber,
		PhotoURL:              in.PhotoURL,
		GuardID:               in.GuardID,
		Source:                entities.VisitorEntrySourceQR,
		Notes:                 in.Notes,
		Status:                entities.VisitorEntryStatusActive,
	})
	if err != nil {
		return entities.VisitorEntry{}, apperrors.Internal("failed to register entry")
	}
	return entry, nil
}

// CheckinManual registra una entrada manual.
//
// Reglas:
//   - PhotoURL OBLIGATORIO.
//   - GuardID UUID valido.
//   - VisitorDocumentNumber requerido (para auditoria + cruce blacklist).
//   - Si DocumentType viene, debe ser valido.
//   - Si UnitID viene, debe ser UUID valido.
//   - Si esta en blacklist activa: registra entrada 'rejected' y devuelve
//     403 + notes con el motivo.
type CheckinManual struct {
	BlacklistRepo domain.BlacklistRepository
	EntryRepo     domain.VisitorEntryRepository
}

// CheckinManualInput es el input del usecase.
type CheckinManualInput struct {
	UnitID                *string
	VisitorFullName       string
	VisitorDocumentType   *string
	VisitorDocumentNumber string
	PhotoURL              string
	GuardID               string
	Notes                 *string
}

// Execute valida y delega.
func (u CheckinManual) Execute(ctx context.Context, in CheckinManualInput) (entities.VisitorEntry, error) {
	if strings.TrimSpace(in.PhotoURL) == "" {
		return entities.VisitorEntry{}, mapPhotoRequired()
	}
	if strings.TrimSpace(in.VisitorFullName) == "" {
		return entities.VisitorEntry{}, apperrors.BadRequest("visitor_full_name is required")
	}
	if err := policies.ValidateDocumentNumber(in.VisitorDocumentNumber); err != nil {
		return entities.VisitorEntry{}, apperrors.BadRequest("visitor_document_number: " + err.Error())
	}
	if err := policies.ValidateUUID(in.GuardID); err != nil {
		return entities.VisitorEntry{}, apperrors.BadRequest("guard_id: " + err.Error())
	}
	if in.UnitID != nil {
		if err := policies.ValidateUUID(*in.UnitID); err != nil {
			return entities.VisitorEntry{}, apperrors.BadRequest("unit_id: " + err.Error())
		}
	}
	if in.VisitorDocumentType != nil {
		if err := policies.ValidateDocumentType(*in.VisitorDocumentType); err != nil {
			return entities.VisitorEntry{}, apperrors.BadRequest("visitor_document_type: " + err.Error())
		}
	}

	// Chequeo de blacklist (si hay tipo de documento).
	if in.VisitorDocumentType != nil && *in.VisitorDocumentType != "" {
		bl, err := u.BlacklistRepo.Get(ctx,
			entities.DocumentType(*in.VisitorDocumentType),
			in.VisitorDocumentNumber)
		if err != nil {
			return entities.VisitorEntry{}, apperrors.Internal("failed to check blacklist")
		}
		if bl != nil {
			notes := fmt.Sprintf("blacklist hit: %s", bl.Reason)
			photo := in.PhotoURL
			_, _ = u.EntryRepo.Create(ctx, domain.CreateVisitorEntryInput{
				UnitID:                in.UnitID,
				PreRegistrationID:     nil,
				VisitorFullName:       strings.TrimSpace(in.VisitorFullName),
				VisitorDocumentType:   in.VisitorDocumentType,
				VisitorDocumentNumber: in.VisitorDocumentNumber,
				PhotoURL:              &photo,
				GuardID:               in.GuardID,
				Source:                entities.VisitorEntrySourceManual,
				Notes:                 &notes,
				Status:                entities.VisitorEntryStatusRejected,
			})
			return entities.VisitorEntry{}, mapBlacklistErr(bl.Reason)
		}
	}

	photo := in.PhotoURL
	entry, err := u.EntryRepo.Create(ctx, domain.CreateVisitorEntryInput{
		UnitID:                in.UnitID,
		PreRegistrationID:     nil,
		VisitorFullName:       strings.TrimSpace(in.VisitorFullName),
		VisitorDocumentType:   in.VisitorDocumentType,
		VisitorDocumentNumber: in.VisitorDocumentNumber,
		PhotoURL:              &photo,
		GuardID:               in.GuardID,
		Source:                entities.VisitorEntrySourceManual,
		Notes:                 in.Notes,
		Status:                entities.VisitorEntryStatusActive,
	})
	if err != nil {
		return entities.VisitorEntry{}, apperrors.Internal("failed to register entry")
	}
	return entry, nil
}

// Checkout cierra una entrada activa fijando exit_time = now.
type Checkout struct {
	EntryRepo domain.VisitorEntryRepository
}

// CheckoutInput es el input del usecase.
type CheckoutInput struct {
	EntryID string
	ActorID string
}

// Execute valida y delega.
func (u Checkout) Execute(ctx context.Context, in CheckoutInput) (entities.VisitorEntry, error) {
	if err := policies.ValidateUUID(in.EntryID); err != nil {
		return entities.VisitorEntry{}, apperrors.BadRequest("entry_id: " + err.Error())
	}
	entry, err := u.EntryRepo.Close(ctx, in.EntryID, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			return entities.VisitorEntry{}, apperrors.NotFound("entry not found or already closed")
		}
		return entities.VisitorEntry{}, apperrors.Internal("failed to close entry")
	}
	return entry, nil
}

// ListActiveVisits lista las visitas activas (sin exit_time).
type ListActiveVisits struct {
	EntryRepo domain.VisitorEntryRepository
}

// Execute delega al repo.
func (u ListActiveVisits) Execute(ctx context.Context) ([]entities.VisitorEntry, error) {
	out, err := u.EntryRepo.ListActive(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list active visits")
	}
	return out, nil
}

func mapBlacklistErr(reason string) error {
	return apperrors.New(403, "blacklisted",
		"Forbidden",
		fmt.Sprintf("visitor is blacklisted: %s", reason))
}

func mapPhotoRequired() error {
	return apperrors.New(400, "photo-required",
		"Bad Request",
		"photo_url is required for manual check-in")
}
