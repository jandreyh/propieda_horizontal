package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo penalties.
type OutboxEventType string

const (
	// OutboxEventPenaltyNotified se emite al notificar una sancion al
	// deudor.
	OutboxEventPenaltyNotified OutboxEventType = "penalty.notified"
	// OutboxEventPenaltyAppealed se emite cuando el deudor apela una
	// sancion.
	OutboxEventPenaltyAppealed OutboxEventType = "penalty.appealed"
	// OutboxEventPenaltyConfirmed se emite al confirmar una sancion.
	OutboxEventPenaltyConfirmed OutboxEventType = "penalty.confirmed"
	// OutboxEventPenaltyDismissed se emite al desestimar una sancion
	// (apelacion aceptada).
	OutboxEventPenaltyDismissed OutboxEventType = "penalty.dismissed"
	// OutboxEventPenaltySettled se emite al saldar una sancion.
	OutboxEventPenaltySettled OutboxEventType = "penalty.settled"
	// OutboxEventChargeRequested se emite al confirmar una sancion
	// monetaria para que el modulo de cargos genere la deuda.
	OutboxEventChargeRequested OutboxEventType = "penalty.charge_requested"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventPenaltyNotified, OutboxEventPenaltyAppealed,
		OutboxEventPenaltyConfirmed, OutboxEventPenaltyDismissed,
		OutboxEventPenaltySettled, OutboxEventChargeRequested:
		return true
	}
	return false
}

// OutboxEvent representa un evento modulo-local pendiente de despacho
// (patron outbox).
type OutboxEvent struct {
	ID             string
	PenaltyID      string
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
