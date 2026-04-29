package entities

import "time"

// OutboxStatus enumera los estados validos de un mensaje en el outbox.
type OutboxStatus string

const (
	// OutboxStatusPending indica que el mensaje esta pendiente de envio.
	OutboxStatusPending OutboxStatus = "pending"
	// OutboxStatusScheduled indica que el mensaje esta programado.
	OutboxStatusScheduled OutboxStatus = "scheduled"
	// OutboxStatusSending indica que el mensaje esta en proceso de envio.
	OutboxStatusSending OutboxStatus = "sending"
	// OutboxStatusSent indica que el mensaje fue enviado exitosamente.
	OutboxStatusSent OutboxStatus = "sent"
	// OutboxStatusFailedRetry indica que el envio fallo y se reintentara.
	OutboxStatusFailedRetry OutboxStatus = "failed_retry"
	// OutboxStatusFailedPermanent indica que el envio fallo
	// definitivamente (max attempts alcanzado).
	OutboxStatusFailedPermanent OutboxStatus = "failed_permanent"
	// OutboxStatusBlockedNoConsent indica que se bloqueo por falta de
	// consentimiento.
	OutboxStatusBlockedNoConsent OutboxStatus = "blocked_no_consent"
	// OutboxStatusCancelled indica que el envio fue cancelado.
	OutboxStatusCancelled OutboxStatus = "cancelled"
)

// IsValid indica si el status es uno de los enumerados.
func (s OutboxStatus) IsValid() bool {
	switch s {
	case OutboxStatusPending, OutboxStatusScheduled, OutboxStatusSending,
		OutboxStatusSent, OutboxStatusFailedRetry, OutboxStatusFailedPermanent,
		OutboxStatusBlockedNoConsent, OutboxStatusCancelled:
		return true
	}
	return false
}

// IsTerminal indica si el status es un estado final (no se puede
// transicionar mas).
func (s OutboxStatus) IsTerminal() bool {
	switch s {
	case OutboxStatusSent, OutboxStatusFailedPermanent,
		OutboxStatusBlockedNoConsent, OutboxStatusCancelled:
		return true
	}
	return false
}

// NotificationOutbox representa un mensaje en la cola de envios
// (patron outbox, at-least-once delivery).
type NotificationOutbox struct {
	ID              string
	EventType       string
	RecipientUserID string
	Channel         Channel
	Payload         []byte
	IdempotencyKey  string
	ScheduledAt     time.Time
	SentAt          *time.Time
	Attempts        int32
	LastError       *string
	Status          OutboxStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	DeletedBy       *string
	Version         int32
}
