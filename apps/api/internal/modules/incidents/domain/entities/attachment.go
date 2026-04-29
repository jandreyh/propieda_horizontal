package entities

import "time"

// Attachment representa un adjunto (foto/video) de un incidente.
type Attachment struct {
	ID         string
	IncidentID string
	URL        string
	MimeType   string
	SizeBytes  int64
	UploadedBy string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	CreatedBy  *string
	UpdatedBy  *string
	DeletedBy  *string
}
