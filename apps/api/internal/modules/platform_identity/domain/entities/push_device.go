package entities

import (
	"time"

	"github.com/google/uuid"
)

// PushDevice es un device token registrado por la persona para recibir
// notificaciones push centralizadas.
type PushDevice struct {
	ID             uuid.UUID
	PlatformUserID uuid.UUID
	DeviceToken    string
	Platform       string // ios | android | web
	DeviceLabel    *string
	LastSeenAt     time.Time
	CreatedAt      time.Time
	RevokedAt      *time.Time
}

// IsActive indica si el device puede recibir notifs.
func (d PushDevice) IsActive() bool {
	return d.RevokedAt == nil
}
