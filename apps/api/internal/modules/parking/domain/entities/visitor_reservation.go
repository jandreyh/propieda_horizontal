package entities

import "time"

// ReservationStatus enumera los estados validos de una reserva de
// visitante.
type ReservationStatus string

const (
	// ReservationStatusPending indica que la reserva esta pendiente de
	// confirmacion.
	ReservationStatusPending ReservationStatus = "pending"
	// ReservationStatusConfirmed indica que la reserva fue confirmada.
	ReservationStatusConfirmed ReservationStatus = "confirmed"
	// ReservationStatusCancelled indica que la reserva fue cancelada.
	ReservationStatusCancelled ReservationStatus = "cancelled"
	// ReservationStatusNoShow indica que el visitante no se presento.
	ReservationStatusNoShow ReservationStatus = "no_show"
	// ReservationStatusConsumed indica que el visitante uso el espacio.
	ReservationStatusConsumed ReservationStatus = "consumed"
)

// IsValid indica si el status es uno de los enumerados.
func (s ReservationStatus) IsValid() bool {
	switch s {
	case ReservationStatusPending, ReservationStatusConfirmed,
		ReservationStatusCancelled, ReservationStatusNoShow,
		ReservationStatusConsumed:
		return true
	}
	return false
}

// VisitorReservation representa una reserva de espacio de parqueadero
// para un visitante, con un slot temporal definido.
type VisitorReservation struct {
	ID              string
	ParkingSpaceID  string
	UnitID          string
	RequestedBy     string
	VisitorName     string
	VisitorDocument *string
	VehiclePlate    *string
	SlotStartAt     time.Time
	SlotEndAt       time.Time
	IdempotencyKey  *string
	Status          ReservationStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	DeletedBy       *string
	Version         int32
}
