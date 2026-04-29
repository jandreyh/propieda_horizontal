package entities

import "time"

// StatusHistory representa un registro append-only de transicion de
// estado de un ticket PQRS.
type StatusHistory struct {
	ID                   string
	TicketID             string
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
