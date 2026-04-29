package entities

import "time"

// NotificationProviderConfig representa la configuracion de un proveedor
// de envio para un canal. Solo puede haber un proveedor activo por canal.
type NotificationProviderConfig struct {
	ID           string
	Channel      Channel
	ProviderName string
	Config       []byte
	IsActive     bool
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}
