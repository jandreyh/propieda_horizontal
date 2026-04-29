package entities

import "time"

// CallStatus enumera los estados validos de una convocatoria.
type CallStatus string

const (
	// CallStatusDraft la convocatoria esta en borrador.
	CallStatusDraft CallStatus = "draft"
	// CallStatusPublished la convocatoria fue publicada.
	CallStatusPublished CallStatus = "published"
	// CallStatusCancelled la convocatoria fue cancelada.
	CallStatusCancelled CallStatus = "cancelled"
	// CallStatusArchived la convocatoria fue archivada.
	CallStatusArchived CallStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s CallStatus) IsValid() bool {
	switch s {
	case CallStatusDraft, CallStatusPublished,
		CallStatusCancelled, CallStatusArchived:
		return true
	}
	return false
}

// AssemblyCall representa una convocatoria formal de asamblea.
type AssemblyCall struct {
	ID          string
	AssemblyID  string
	PublishedAt time.Time
	Channels    []byte
	Agenda      []byte
	BodyMD      *string
	PublishedBy *string
	Status      CallStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}
