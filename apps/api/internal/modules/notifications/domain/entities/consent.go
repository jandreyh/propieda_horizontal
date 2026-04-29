package entities

import "time"

// ConsentStatus enumera los estados validos de un consentimiento.
type ConsentStatus string

const (
	// ConsentStatusActive indica que el consentimiento esta vigente.
	ConsentStatusActive ConsentStatus = "active"
	// ConsentStatusRevoked indica que el consentimiento fue revocado.
	ConsentStatusRevoked ConsentStatus = "revoked"
)

// IsValid indica si el status es uno de los enumerados.
func (s ConsentStatus) IsValid() bool {
	switch s {
	case ConsentStatusActive, ConsentStatusRevoked:
		return true
	}
	return false
}

// NotificationConsent representa el consentimiento legal de un usuario
// para recibir notificaciones por un canal (opt-in).
type NotificationConsent struct {
	ID              string
	UserID          string
	Channel         Channel
	ConsentedAt     time.Time
	RevokedAt       *time.Time
	ConsentProofURL *string
	LegalBasis      *string
	Status          ConsentStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	DeletedBy       *string
	Version         int32
}

// IsActive indica si el consentimiento esta vigente y no revocado.
func (c NotificationConsent) IsActive() bool {
	return c.Status == ConsentStatusActive && c.RevokedAt == nil && c.DeletedAt == nil
}
