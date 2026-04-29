package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo assemblies.
type OutboxEventType string

const (
	// OutboxEventAssemblyCreated se emite al crear una asamblea.
	OutboxEventAssemblyCreated OutboxEventType = "assembly.created"
	// OutboxEventAssemblyCalled se emite al publicar la convocatoria.
	OutboxEventAssemblyCalled OutboxEventType = "assembly.called"
	// OutboxEventAssemblyStarted se emite al iniciar la asamblea.
	OutboxEventAssemblyStarted OutboxEventType = "assembly.started"
	// OutboxEventAssemblyClosed se emite al cerrar la asamblea.
	OutboxEventAssemblyClosed OutboxEventType = "assembly.closed"
	// OutboxEventMotionOpened se emite al abrir votacion de una mocion.
	OutboxEventMotionOpened OutboxEventType = "assembly.motion_opened"
	// OutboxEventMotionClosed se emite al cerrar votacion de una mocion.
	OutboxEventMotionClosed OutboxEventType = "assembly.motion_closed"
	// OutboxEventVoteCast se emite al registrar un voto.
	OutboxEventVoteCast OutboxEventType = "assembly.vote_cast"
	// OutboxEventActSigned se emite al firmar un acta.
	OutboxEventActSigned OutboxEventType = "assembly.act_signed"
)

// IsValid indica si el tipo de evento es uno de los enumerados.
func (t OutboxEventType) IsValid() bool {
	switch t {
	case OutboxEventAssemblyCreated, OutboxEventAssemblyCalled,
		OutboxEventAssemblyStarted, OutboxEventAssemblyClosed,
		OutboxEventMotionOpened, OutboxEventMotionClosed,
		OutboxEventVoteCast, OutboxEventActSigned:
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
