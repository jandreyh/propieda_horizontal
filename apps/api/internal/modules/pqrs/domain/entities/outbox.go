package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo pqrs.
type OutboxEventType string

const (
	// OutboxEventPQRSCreated se emite al crear un ticket.
	OutboxEventPQRSCreated OutboxEventType = "pqrs.created"
	// OutboxEventPQRSAssigned se emite al asignar un ticket.
	OutboxEventPQRSAssigned OutboxEventType = "pqrs.assigned"
	// OutboxEventPQRSResponded se emite al responder oficialmente un
	// ticket.
	OutboxEventPQRSResponded OutboxEventType = "pqrs.responded"
	// OutboxEventPQRSClosed se emite al cerrar un ticket.
	OutboxEventPQRSClosed OutboxEventType = "pqrs.closed"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventPQRSCreated, OutboxEventPQRSAssigned,
		OutboxEventPQRSResponded, OutboxEventPQRSClosed:
		return true
	}
	return false
}

// OutboxEvent representa un evento modulo-local pendiente de despacho
// (patron outbox).
type OutboxEvent struct {
	ID             string
	TicketID       string
	EventType      OutboxEventType
	Payload        []byte
	IdempotencyKey *string
	CreatedAt      time.Time
	NextAttemptAt  time.Time
	Attempts       int32
	DeliveredAt    *time.Time
	LastError      *string
}

// IsPending indica si el evento sigue pendiente de despacho.
func (e OutboxEvent) IsPending() bool {
	return e.DeliveredAt == nil
}
