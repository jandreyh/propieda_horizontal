// Package domain define los puertos del modulo access_control.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
)

// --- Sentinels comunes ---

// ErrPreregistrationNotFound se devuelve cuando el QR no se ubica.
var ErrPreregistrationNotFound = errors.New("access_control: pre-registration not found")

// ErrPreregistrationExhausted se devuelve cuando el QR ya consumio todos
// sus usos. La capa HTTP la mapea a 410 Gone.
var ErrPreregistrationExhausted = errors.New("access_control: pre-registration exhausted")

// ErrPreregistrationExpired se devuelve cuando el QR ya paso expires_at.
// La capa HTTP la mapea a 410 Gone.
var ErrPreregistrationExpired = errors.New("access_control: pre-registration expired")

// ErrBlacklisted se devuelve cuando el documento del visitante esta en
// la blacklist activa. La capa HTTP la mapea a 403 Forbidden y registra
// la entrada como 'rejected' por auditoria.
var ErrBlacklisted = errors.New("access_control: visitor is blacklisted")

// ErrPhotoRequired se devuelve cuando un checkin manual llega sin
// photo_url (campo obligatorio en ese flujo). Mapea a 400.
var ErrPhotoRequired = errors.New("access_control: photo_url is required for manual check-in")

// ErrEntryNotFound se devuelve cuando una entrada por id no existe.
var ErrEntryNotFound = errors.New("access_control: visitor entry not found")

// ErrBlacklistNotFound se devuelve cuando una entrada de blacklist por
// id no existe (o esta archivada).
var ErrBlacklistNotFound = errors.New("access_control: blacklist entry not found")

// ErrBlacklistAlreadyExists se devuelve cuando ya existe una entrada
// activa para (document_type, document_number).
var ErrBlacklistAlreadyExists = errors.New("access_control: blacklist entry already exists")

// --- BlacklistRepository ---

// CreateBlacklistInput agrupa los datos necesarios para crear una entrada
// de blacklist.
type CreateBlacklistInput struct {
	DocumentType     entities.DocumentType
	DocumentNumber   string
	FullName         *string
	Reason           string
	ReportedByUnitID *string
	ReportedByUserID *string
	ExpiresAt        *time.Time
	ActorID          string
}

// BlacklistRepository es el puerto que persiste personas vetadas.
type BlacklistRepository interface {
	// Get devuelve la entrada activa (no archivada y no expirada) para
	// (documentType, documentNumber). Si no hay match, devuelve nil sin
	// error: una "ausencia" no es un error en el camino caliente.
	Get(ctx context.Context, documentType entities.DocumentType, documentNumber string) (*entities.BlacklistEntry, error)
	// Create inserta una nueva entrada. Si choca con la UNIQUE parcial,
	// devuelve ErrBlacklistAlreadyExists.
	Create(ctx context.Context, in CreateBlacklistInput) (entities.BlacklistEntry, error)
	// List devuelve todas las entradas activas (no eliminadas).
	List(ctx context.Context) ([]entities.BlacklistEntry, error)
	// Archive marca una entrada como soft-deleted. Devuelve
	// ErrBlacklistNotFound si no existe o ya esta archivada.
	Archive(ctx context.Context, id, actorID string) (entities.BlacklistEntry, error)
}

// --- PreRegistrationRepository ---

// CreatePreRegistrationInput agrupa los datos para persistir un
// pre-registro nuevo. El QRCodeHash llega ya hasheado (sha256) por la
// capa de aplicacion; el plano NUNCA se persiste.
type CreatePreRegistrationInput struct {
	UnitID                string
	CreatedByUserID       string
	VisitorFullName       string
	VisitorDocumentType   *string
	VisitorDocumentNumber *string
	ExpectedAt            *time.Time
	ExpiresAt             time.Time
	MaxUses               int32
	QRCodeHash            string
}

// PreRegistrationRepository es el puerto que persiste pre-registros.
type PreRegistrationRepository interface {
	// Create inserta un pre-registro nuevo.
	Create(ctx context.Context, in CreatePreRegistrationInput) (entities.PreRegistration, error)
	// GetByQRHash devuelve un pre-registro por hash (cualquier estado).
	// Devuelve ErrPreregistrationNotFound si no existe.
	GetByQRHash(ctx context.Context, qrHash string) (entities.PreRegistration, error)
	// ConsumeOne consume UN uso del pre-registro identificado por qrHash
	// de manera atomica. Si afecta 0 filas (expirado, agotado, revocado),
	// devuelve ErrPreregistrationExhausted o ErrPreregistrationNotFound
	// segun corresponda al diagnostico.
	ConsumeOne(ctx context.Context, qrHash string) (entities.PreRegistration, error)
}

// --- VisitorEntryRepository ---

// CreateVisitorEntryInput agrupa los datos para persistir una entrada
// nueva (sea valida o rechazada).
type CreateVisitorEntryInput struct {
	UnitID                *string
	PreRegistrationID     *string
	VisitorFullName       string
	VisitorDocumentType   *string
	VisitorDocumentNumber string
	PhotoURL              *string
	GuardID               string
	Source                entities.VisitorEntrySource
	Notes                 *string
	Status                entities.VisitorEntryStatus
}

// VisitorEntryRepository es el puerto que persiste entradas de
// visitantes.
type VisitorEntryRepository interface {
	// Create inserta una entrada (active o rejected).
	Create(ctx context.Context, in CreateVisitorEntryInput) (entities.VisitorEntry, error)
	// Close fija exit_time = now() en una entrada activa. Devuelve
	// ErrEntryNotFound si no existe o no esta activa.
	Close(ctx context.Context, entryID, actorID string) (entities.VisitorEntry, error)
	// ListActive devuelve las entradas con exit_time = NULL y status =
	// 'active', ordenadas por entry_time desc. Para el dashboard del
	// guarda.
	ListActive(ctx context.Context) ([]entities.VisitorEntry, error)
	// GetByID devuelve una entrada por id (cualquier estado). Devuelve
	// ErrEntryNotFound si no existe.
	GetByID(ctx context.Context, id string) (entities.VisitorEntry, error)
}
