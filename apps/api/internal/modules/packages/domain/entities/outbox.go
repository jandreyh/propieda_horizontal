package entities

import "time"

// OutboxEventType es la lista cerrada de eventos del modulo.
type OutboxEventType string

const (
	// OutboxEventPackageReceived se emite al crear un paquete (notifica al
	// residente que tiene un paquete en porteria).
	OutboxEventPackageReceived OutboxEventType = "package.received"
	// OutboxEventPackageDelivered se emite al entregar el paquete.
	OutboxEventPackageDelivered OutboxEventType = "package.delivered"
	// OutboxEventPackageReminder se emite por el cron diario cuando el
	// paquete lleva mas de 3 dias en porteria.
	OutboxEventPackageReminder OutboxEventType = "package.reminder"
)

// OutboxEvent representa un evento modulo-local pendiente de despacho.
type OutboxEvent struct {
	ID            string
	PackageID     string
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
