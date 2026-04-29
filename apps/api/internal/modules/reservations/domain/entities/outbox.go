package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo reservations.
type OutboxEventType string

const (
	// OutboxEventReservationCreated se emite al crear una reserva.
	OutboxEventReservationCreated OutboxEventType = "reservation.created"
	// OutboxEventReservationConfirmed se emite al confirmar (aprobar)
	// una reserva.
	OutboxEventReservationConfirmed OutboxEventType = "reservation.confirmed"
	// OutboxEventReservationCancelled se emite al cancelar una reserva.
	OutboxEventReservationCancelled OutboxEventType = "reservation.cancelled"
	// OutboxEventReservationRejected se emite al rechazar una reserva.
	OutboxEventReservationRejected OutboxEventType = "reservation.rejected"
	// OutboxEventReservationConsumed se emite al registrar checkin
	// (consumo) de una reserva.
	OutboxEventReservationConsumed OutboxEventType = "reservation.consumed"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventReservationCreated, OutboxEventReservationConfirmed,
		OutboxEventReservationCancelled, OutboxEventReservationRejected,
		OutboxEventReservationConsumed:
		return true
	}
	return false
}

// OutboxEvent representa un evento modulo-local pendiente de despacho
// (patron outbox).
type OutboxEvent struct {
	ID            string
	AggregateID   string
	EventType     OutboxEventType
	Payload       []byte
	CreatedAt     time.Time
	NextAttemptAt time.Time
	Attempts      int32
	DeliveredAt   *time.Time
	LastError     *string
}

// IsPending indica si el evento sigue pendiente de despacho.
func (e OutboxEvent) IsPending() bool {
	return e.DeliveredAt == nil
}
