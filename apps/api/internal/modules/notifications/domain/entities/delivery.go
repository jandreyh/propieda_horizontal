package entities

import "time"

// DeliveryStatus enumera los estados validos de un registro de entrega.
type DeliveryStatus string

const (
	// DeliveryStatusSubmitted indica que fue enviado al proveedor.
	DeliveryStatusSubmitted DeliveryStatus = "submitted"
	// DeliveryStatusDelivered indica entrega confirmada por el proveedor.
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	// DeliveryStatusFailed indica fallo reportado por el proveedor.
	DeliveryStatusFailed DeliveryStatus = "failed"
	// DeliveryStatusUnknown indica estado desconocido.
	DeliveryStatusUnknown DeliveryStatus = "unknown"
)

// IsValid indica si el status es uno de los enumerados.
func (s DeliveryStatus) IsValid() bool {
	switch s {
	case DeliveryStatusSubmitted, DeliveryStatusDelivered,
		DeliveryStatusFailed, DeliveryStatusUnknown:
		return true
	}
	return false
}

// NotificationDelivery representa un registro de entrega por proveedor
// asociado a un mensaje del outbox.
type NotificationDelivery struct {
	ID                string
	OutboxID          string
	ProviderName      string
	ProviderMessageID *string
	ProviderStatus    *string
	DeliveredAt       *time.Time
	FailureReason     *string
	Status            DeliveryStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
}
