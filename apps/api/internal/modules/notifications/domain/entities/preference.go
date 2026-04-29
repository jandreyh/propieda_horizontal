package entities

import "time"

// PreferenceStatus enumera los estados validos de una preferencia.
type PreferenceStatus string

const (
	// PreferenceStatusActive indica que la preferencia esta activa.
	PreferenceStatusActive PreferenceStatus = "active"
	// PreferenceStatusArchived indica que la preferencia fue archivada.
	PreferenceStatusArchived PreferenceStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s PreferenceStatus) IsValid() bool {
	switch s {
	case PreferenceStatusActive, PreferenceStatusArchived:
		return true
	}
	return false
}

// NotificationPreference representa la preferencia de un usuario para
// un evento en un canal especifico.
type NotificationPreference struct {
	ID        string
	UserID    string
	EventType string
	Channel   Channel
	Enabled   bool
	Status    PreferenceStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int32
}
