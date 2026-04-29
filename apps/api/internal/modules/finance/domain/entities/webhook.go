package entities

import "time"

// WebhookIdempotency representa un registro de deduplicacion de webhook
// de pasarela de pago.
type WebhookIdempotency struct {
	ID             string
	Gateway        string
	IdempotencyKey string
	PayloadHash    *string
	ReceivedAt     time.Time
	ProcessedAt    *time.Time
	PaymentID      *string
	LastError      *string
}

// IsProcessed indica si el webhook ya fue procesado exitosamente.
func (w WebhookIdempotency) IsProcessed() bool {
	return w.ProcessedAt != nil && w.LastError == nil
}
