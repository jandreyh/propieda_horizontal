package entities

import "time"

// ReservationStatusHistory representa un registro append-only de cambio
// de estado de una reserva.
type ReservationStatusHistory struct {
	ID            string
	ReservationID string
	FromStatus    *string
	ToStatus      string
	ChangedBy     *string
	Reason        *string
	ChangedAt     time.Time
}
