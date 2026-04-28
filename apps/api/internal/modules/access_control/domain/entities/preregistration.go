package entities

import "time"

// PreRegistrationStatus enumera los estados validos de un pre-registro
// de visitante.
type PreRegistrationStatus string

const (
	// PreRegistrationStatusActive es el QR sigue siendo redimible.
	PreRegistrationStatusActive PreRegistrationStatus = "active"
	// PreRegistrationStatusExpired es paso expires_at sin redimirse del todo.
	PreRegistrationStatusExpired PreRegistrationStatus = "expired"
	// PreRegistrationStatusConsumed es agotados los max_uses.
	PreRegistrationStatusConsumed PreRegistrationStatus = "consumed"
	// PreRegistrationStatusRevoked es el residente revoco el QR.
	PreRegistrationStatusRevoked PreRegistrationStatus = "revoked"
)

// PreRegistration representa un pre-registro de visitante con QR firmado.
//
// El QR plano NO se persiste; solo su sha256 (`QRCodeHash`). El plano se
// devuelve UNA SOLA VEZ al residente que lo crea.
type PreRegistration struct {
	ID                    string
	UnitID                string
	CreatedByUserID       string
	VisitorFullName       string
	VisitorDocumentType   *string
	VisitorDocumentNumber *string
	ExpectedAt            *time.Time
	ExpiresAt             time.Time
	MaxUses               int32
	UsesCount             int32
	QRCodeHash            string
	Status                PreRegistrationStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
	CreatedBy             *string
	UpdatedBy             *string
	DeletedBy             *string
	Version               int32
}

// IsActive indica si sigue redimible (estado activo y sin soft-delete).
func (p PreRegistration) IsActive() bool {
	return p.Status == PreRegistrationStatusActive && p.DeletedAt == nil
}
