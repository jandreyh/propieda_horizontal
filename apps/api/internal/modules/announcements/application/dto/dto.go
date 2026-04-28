// Package dto contiene los Data Transfer Objects del modulo
// announcements. Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// AudienceTarget describe una audiencia destinataria del anuncio. Para
// `global`, TargetID DEBE ser nil; para los demas tipos, es obligatorio.
type AudienceTarget struct {
	TargetType string  `json:"target_type"`
	TargetID   *string `json:"target_id,omitempty"`
}

// CreateAnnouncementRequest es el body de POST /announcements.
type CreateAnnouncementRequest struct {
	Title             string           `json:"title"`
	Body              string           `json:"body"`
	PublishedByUserID string           `json:"published_by_user_id,omitempty"`
	Pinned            *bool            `json:"pinned,omitempty"`
	ExpiresAt         *time.Time       `json:"expires_at,omitempty"`
	Audiences         []AudienceTarget `json:"audiences"`
}

// AnnouncementResponse es la representacion HTTP de un Announcement
// junto con sus audiencias.
type AnnouncementResponse struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	PublishedBy string           `json:"published_by"`
	PublishedAt time.Time        `json:"published_at"`
	Pinned      bool             `json:"pinned"`
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
	Status      string           `json:"status"`
	Audiences   []AudienceTarget `json:"audiences"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Version     int32            `json:"version"`
}

// FeedRequest agrupa los parametros de la consulta del feed.
type FeedRequest struct {
	UserID       string   `json:"user_id"`
	RoleIDs      []string `json:"role_ids"`
	StructureIDs []string `json:"structure_ids"`
	UnitIDs      []string `json:"unit_ids"`
	Limit        int32    `json:"limit"`
	Offset       int32    `json:"offset"`
}

// FeedResponse es el sobre del feed.
type FeedResponse struct {
	Items []AnnouncementResponse `json:"items"`
	Total int                    `json:"total"`
}

// AckResponse es la representacion HTTP de una confirmacion.
type AckResponse struct {
	ID             string    `json:"id"`
	AnnouncementID string    `json:"announcement_id"`
	UserID         string    `json:"user_id"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}
