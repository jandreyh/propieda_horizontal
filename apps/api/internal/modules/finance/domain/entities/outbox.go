package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo finance.
type OutboxEventType string

const (
	// OutboxEventChargeCreated se emite al crear un cargo.
	OutboxEventChargeCreated OutboxEventType = "finance.charge_created"
	// OutboxEventPaymentCaptured se emite al capturar un pago.
	OutboxEventPaymentCaptured OutboxEventType = "finance.payment_captured"
	// OutboxEventPaymentAllocated se emite al aplicar un pago a cargos.
	OutboxEventPaymentAllocated OutboxEventType = "finance.payment_allocated"
	// OutboxEventPaymentReversed se emite al completar un reverso.
	OutboxEventPaymentReversed OutboxEventType = "finance.payment_reversed"
	// OutboxEventPeriodClosedSoft se emite al cerrar un periodo soft.
	OutboxEventPeriodClosedSoft OutboxEventType = "finance.period_closed_soft"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventChargeCreated, OutboxEventPaymentCaptured,
		OutboxEventPaymentAllocated, OutboxEventPaymentReversed,
		OutboxEventPeriodClosedSoft:
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
