package entities

import "time"

// ResponseType enumera los tipos validos de respuesta a un ticket PQRS.
type ResponseType string

const (
	// ResponseTypeInternalNote es una nota interna (no visible al
	// residente).
	ResponseTypeInternalNote ResponseType = "internal_note"
	// ResponseTypeOfficialResponse es la respuesta oficial al
	// residente.
	ResponseTypeOfficialResponse ResponseType = "official_response"
)

// IsValid indica si el tipo es uno de los enumerados.
func (t ResponseType) IsValid() bool {
	switch t {
	case ResponseTypeInternalNote, ResponseTypeOfficialResponse:
		return true
	}
	return false
}

// Response representa una respuesta o nota interna asociada a un
// ticket PQRS.
type Response struct {
	ID                string
	TicketID          string
	ResponseType      ResponseType
	Body              string
	RespondedByUserID string
	RespondedAt       time.Time
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}
