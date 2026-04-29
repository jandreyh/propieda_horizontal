package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo incidents.
type OutboxEventType string

const (
	// OutboxEventIncidentReported se emite al reportar un incidente.
	OutboxEventIncidentReported OutboxEventType = "incident.reported"
	// OutboxEventIncidentAssigned se emite al asignar un incidente.
	OutboxEventIncidentAssigned OutboxEventType = "incident.assigned"
	// OutboxEventIncidentEscalated se emite cuando un incidente es
	// escalado por incumplimiento de SLA.
	OutboxEventIncidentEscalated OutboxEventType = "incident.escalated"
	// OutboxEventIncidentResolved se emite al resolver un incidente.
	OutboxEventIncidentResolved OutboxEventType = "incident.resolved"
	// OutboxEventIncidentClosed se emite al cerrar un incidente.
	OutboxEventIncidentClosed OutboxEventType = "incident.closed"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventIncidentReported, OutboxEventIncidentAssigned,
		OutboxEventIncidentEscalated, OutboxEventIncidentResolved,
		OutboxEventIncidentClosed:
		return true
	}
	return false
}

// OutboxEvent representa un evento modulo-local pendiente de despacho
// (patron outbox, ADR 0005).
type OutboxEvent struct {
	ID            string
	IncidentID    string
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
