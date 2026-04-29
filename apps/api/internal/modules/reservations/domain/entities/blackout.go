package entities

import "time"

// BlackoutStatus enumera los estados validos de un bloqueo.
type BlackoutStatus string

const (
	// BlackoutStatusActive indica que el bloqueo esta vigente.
	BlackoutStatusActive BlackoutStatus = "active"
	// BlackoutStatusCancelled indica que el bloqueo fue cancelado.
	BlackoutStatusCancelled BlackoutStatus = "cancelled"
	// BlackoutStatusArchived indica que el bloqueo fue archivado.
	BlackoutStatusArchived BlackoutStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s BlackoutStatus) IsValid() bool {
	switch s {
	case BlackoutStatusActive, BlackoutStatusCancelled, BlackoutStatusArchived:
		return true
	}
	return false
}

// ReservationBlackout representa un bloqueo temporal de una zona comun
// (por mantenimiento, asamblea, etc).
type ReservationBlackout struct {
	ID           string
	CommonAreaID string
	FromAt       time.Time
	ToAt         time.Time
	Reason       string
	Status       BlackoutStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}

// IsActive indica si el bloqueo esta vigente.
func (b ReservationBlackout) IsActive() bool {
	return b.Status == BlackoutStatusActive && b.DeletedAt == nil
}
