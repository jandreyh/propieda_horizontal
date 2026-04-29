// Package entities define las entidades de dominio del modulo pqrs.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// PQRType enumera los tipos validos de un ticket PQRS.
type PQRType string

const (
	// PQRTypePeticion es una peticion formal.
	PQRTypePeticion PQRType = "peticion"
	// PQRTypeQueja es una queja.
	PQRTypeQueja PQRType = "queja"
	// PQRTypeReclamo es un reclamo.
	PQRTypeReclamo PQRType = "reclamo"
	// PQRTypeSugerencia es una sugerencia.
	PQRTypeSugerencia PQRType = "sugerencia"
	// PQRTypeSolicitudDocumental es una solicitud de documentos.
	PQRTypeSolicitudDocumental PQRType = "solicitud_documental"
)

// IsValid indica si el tipo es uno de los enumerados.
func (t PQRType) IsValid() bool {
	switch t {
	case PQRTypePeticion, PQRTypeQueja, PQRTypeReclamo,
		PQRTypeSugerencia, PQRTypeSolicitudDocumental:
		return true
	}
	return false
}

// TicketStatus enumera los estados validos de un ticket PQRS.
type TicketStatus string

const (
	// TicketStatusRadicado indica que el ticket fue radicado.
	TicketStatusRadicado TicketStatus = "radicado"
	// TicketStatusEnEstudio indica que el ticket esta en estudio.
	TicketStatusEnEstudio TicketStatus = "en_estudio"
	// TicketStatusRespondido indica que el ticket fue respondido.
	TicketStatusRespondido TicketStatus = "respondido"
	// TicketStatusCerrado indica que el ticket fue cerrado.
	TicketStatusCerrado TicketStatus = "cerrado"
	// TicketStatusEscalado indica que el ticket fue escalado.
	TicketStatusEscalado TicketStatus = "escalado"
	// TicketStatusCancelado indica que el ticket fue cancelado.
	TicketStatusCancelado TicketStatus = "cancelado"
)

// IsValid indica si el status es uno de los enumerados.
func (s TicketStatus) IsValid() bool {
	switch s {
	case TicketStatusRadicado, TicketStatusEnEstudio,
		TicketStatusRespondido, TicketStatusCerrado,
		TicketStatusEscalado, TicketStatusCancelado:
		return true
	}
	return false
}

// Ticket representa un ticket PQRS radicado por un residente.
type Ticket struct {
	ID                string
	TicketYear        int32
	SerialNumber      int32
	PQRType           PQRType
	CategoryID        *string
	Subject           string
	Body              string
	RequesterUserID   string
	AssignedToUserID  *string
	AssignedAt        *time.Time
	RespondedAt       *time.Time
	ClosedAt          *time.Time
	EscalatedAt       *time.Time
	CancelledAt       *time.Time
	SLADueAt          *time.Time
	RequesterRating   *int32
	RequesterFeedback *string
	IsAnonymous       bool
	Status            TicketStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}

// IsOpen indica si el ticket esta abierto (no cerrado/cancelado).
func (t Ticket) IsOpen() bool {
	return t.Status != TicketStatusCerrado &&
		t.Status != TicketStatusCancelado &&
		t.DeletedAt == nil
}
