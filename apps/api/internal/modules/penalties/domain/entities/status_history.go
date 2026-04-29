package entities

import "time"

// PenaltyStatusHistory representa un registro append-only del historial
// de transiciones de estado de una sancion.
type PenaltyStatusHistory struct {
	ID                   string
	PenaltyID            string
	FromStatus           *string
	ToStatus             string
	TransitionedByUserID string
	TransitionedAt       time.Time
	Notes                *string
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CreatedBy            *string
	UpdatedBy            *string
}
