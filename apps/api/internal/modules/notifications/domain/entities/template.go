package entities

import "time"

// TemplateStatus enumera los estados validos de una plantilla.
type TemplateStatus string

const (
	// TemplateStatusActive indica que la plantilla esta activa.
	TemplateStatusActive TemplateStatus = "active"
	// TemplateStatusArchived indica que la plantilla fue archivada.
	TemplateStatusArchived TemplateStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s TemplateStatus) IsValid() bool {
	switch s {
	case TemplateStatusActive, TemplateStatusArchived:
		return true
	}
	return false
}

// NotificationTemplate representa una plantilla de notificacion por
// (event_type, channel, locale).
type NotificationTemplate struct {
	ID                  string
	EventType           string
	Channel             Channel
	Locale              string
	Subject             *string
	BodyTemplate        string
	ProviderTemplateRef *string
	Status              TemplateStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	CreatedBy           *string
	UpdatedBy           *string
	DeletedBy           *string
	Version             int32
}
