package entities

import "time"

// Ack representa la confirmacion de lectura de un anuncio por un
// usuario. El registro es append-only e idempotente (UNIQUE en
// (announcement_id, user_id) + ON CONFLICT DO NOTHING en el insert).
type Ack struct {
	ID             string
	AnnouncementID string
	UserID         string
	AcknowledgedAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      *string
	UpdatedBy      *string
}
