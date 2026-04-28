package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo parking.
type OutboxEventType string

const (
	// OutboxEventParkingAssigned se emite al asignar un espacio a una
	// unidad.
	OutboxEventParkingAssigned OutboxEventType = "parking.assigned"
	// OutboxEventParkingReleased se emite al liberar un espacio
	// (cierre de asignacion).
	OutboxEventParkingReleased OutboxEventType = "parking.released"
	// OutboxEventVisitorReservationCreated se emite al crear una reserva
	// de visitante.
	OutboxEventVisitorReservationCreated OutboxEventType = "parking.visitor_reservation_created"
	// OutboxEventVisitorReservationExpiring se emite cuando una reserva
	// de visitante esta proxima a expirar.
	OutboxEventVisitorReservationExpiring OutboxEventType = "parking.visitor_reservation_expiring"
	// OutboxEventLotteryPublished se emite al publicar los resultados de
	// un sorteo.
	OutboxEventLotteryPublished OutboxEventType = "parking.lottery_published"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventParkingAssigned, OutboxEventParkingReleased,
		OutboxEventVisitorReservationCreated,
		OutboxEventVisitorReservationExpiring,
		OutboxEventLotteryPublished:
		return true
	}
	return false
}

// OutboxEvent representa un evento modulo-local pendiente de despacho
// (patron outbox, ADR 0005).
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
