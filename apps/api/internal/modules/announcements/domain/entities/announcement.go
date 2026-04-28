// Package entities define las entidades de dominio del modulo announcements.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos; nunca aparece como
//     columna ni como campo de dominio.
package entities

import "time"

// AnnouncementStatus enumera los estados validos de un Announcement.
type AnnouncementStatus string

const (
	// StatusPublished marca el anuncio como visible al feed.
	StatusPublished AnnouncementStatus = "published"
	// StatusArchived marca el anuncio como soft-deleted.
	StatusArchived AnnouncementStatus = "archived"
)

// Announcement representa un anuncio del tablero. La capa de aplicacion
// valida title/body antes de pedir su persistencia.
type Announcement struct {
	ID                string
	Title             string
	Body              string
	PublishedByUserID string
	PublishedAt       time.Time
	Pinned            bool
	// ExpiresAt es opcional. Cuando es nil el anuncio no expira.
	ExpiresAt *time.Time
	Status    AnnouncementStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int32
}

// IsArchived indica si el anuncio esta soft-deleted.
func (a Announcement) IsArchived() bool {
	return a.Status == StatusArchived || a.DeletedAt != nil
}
