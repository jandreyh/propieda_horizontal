package entities

import "time"

// ReservationStatus enumera los estados validos de una reserva de zona
// comun.
type ReservationStatus string

const (
	// ReservationStatusPending indica que la reserva esta pendiente de
	// aprobacion.
	ReservationStatusPending ReservationStatus = "pending"
	// ReservationStatusConfirmed indica que la reserva fue confirmada.
	ReservationStatusConfirmed ReservationStatus = "confirmed"
	// ReservationStatusCancelled indica que la reserva fue cancelada.
	ReservationStatusCancelled ReservationStatus = "cancelled"
	// ReservationStatusConsumed indica que la reserva fue usada (checkin).
	ReservationStatusConsumed ReservationStatus = "consumed"
	// ReservationStatusNoShow indica que el reservante no se presento.
	ReservationStatusNoShow ReservationStatus = "no_show"
	// ReservationStatusRejected indica que la reserva fue rechazada.
	ReservationStatusRejected ReservationStatus = "rejected"
	// ReservationStatusArchived indica que la reserva fue archivada.
	ReservationStatusArchived ReservationStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s ReservationStatus) IsValid() bool {
	switch s {
	case ReservationStatusPending, ReservationStatusConfirmed,
		ReservationStatusCancelled, ReservationStatusConsumed,
		ReservationStatusNoShow, ReservationStatusRejected,
		ReservationStatusArchived:
		return true
	}
	return false
}

// Reservation representa una reserva de una zona comun.
type Reservation struct {
	ID                string
	CommonAreaID      string
	UnitID            string
	RequestedByUserID string
	SlotStartAt       time.Time
	SlotEndAt         time.Time
	AttendeesCount    *int32
	Cost              float64
	SecurityDeposit   float64
	DepositRefunded   bool
	QRCodeHash        *string
	IdempotencyKey    *string
	Notes             *string
	ApprovedBy        *string
	ApprovedAt        *time.Time
	CancelledBy       *string
	CancelledAt       *time.Time
	ConsumedAt        *time.Time
	Status            ReservationStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}
