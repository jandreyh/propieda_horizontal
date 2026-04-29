package entities

import "time"

// PushTokenStatus enumera los estados validos de un push token.
type PushTokenStatus string

const (
	// PushTokenStatusActive indica que el token esta activo.
	PushTokenStatusActive PushTokenStatus = "active"
	// PushTokenStatusInvalid indica que el token fue invalidado por el
	// proveedor.
	PushTokenStatusInvalid PushTokenStatus = "invalid"
	// PushTokenStatusArchived indica que el token fue archivado.
	PushTokenStatusArchived PushTokenStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s PushTokenStatus) IsValid() bool {
	switch s {
	case PushTokenStatusActive, PushTokenStatusInvalid, PushTokenStatusArchived:
		return true
	}
	return false
}

// NotificationPushToken representa un token de push registrado para un
// usuario en una plataforma especifica.
type NotificationPushToken struct {
	ID         string
	UserID     string
	Platform   Platform
	Token      string
	LastSeenAt time.Time
	Status     PushTokenStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	CreatedBy  *string
	UpdatedBy  *string
	DeletedBy  *string
	Version    int32
}
