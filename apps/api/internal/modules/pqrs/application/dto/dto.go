// Package dto contiene los Data Transfer Objects del modulo pqrs.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

// CreateCategoryRequest es el body de POST /pqrs/categories.
type CreateCategoryRequest struct {
	Code                  string  `json:"code"`
	Name                  string  `json:"name"`
	DefaultAssigneeRoleID *string `json:"default_assignee_role_id,omitempty"`
}

// UpdateCategoryRequest es el body de PATCH /pqrs/categories/{id}.
type UpdateCategoryRequest struct {
	Code                  string  `json:"code"`
	Name                  string  `json:"name"`
	DefaultAssigneeRoleID *string `json:"default_assignee_role_id,omitempty"`
	Status                string  `json:"status"`
	Version               int32   `json:"version"`
}

// CategoryResponse es la representacion HTTP de una Category.
type CategoryResponse struct {
	ID                    string  `json:"id"`
	Code                  string  `json:"code"`
	Name                  string  `json:"name"`
	DefaultAssigneeRoleID *string `json:"default_assignee_role_id,omitempty"`
	Status                string  `json:"status"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
	Version               int32   `json:"version"`
}

// ListCategoriesResponse es el sobre del listado de categorias.
type ListCategoriesResponse struct {
	Items []CategoryResponse `json:"items"`
	Total int                `json:"total"`
}

// ---------------------------------------------------------------------------
// Tickets
// ---------------------------------------------------------------------------

// CreateTicketRequest es el body de POST /pqrs.
type CreateTicketRequest struct {
	PQRType     string  `json:"pqr_type"`
	CategoryID  *string `json:"category_id,omitempty"`
	Subject     string  `json:"subject"`
	Body        string  `json:"body"`
	IsAnonymous bool    `json:"is_anonymous"`
}

// AssignTicketRequest es el body de POST /pqrs/{id}/assign.
type AssignTicketRequest struct {
	AssignedToUserID string `json:"assigned_to_user_id"`
}

// RespondTicketRequest es el body de POST /pqrs/{id}/respond.
type RespondTicketRequest struct {
	Body string `json:"body"`
}

// AddNoteRequest es el body de POST /pqrs/{id}/notes.
type AddNoteRequest struct {
	Body string `json:"body"`
}

// CloseTicketRequest es el body de POST /pqrs/{id}/close.
type CloseTicketRequest struct {
	Rating   *int32  `json:"rating,omitempty"`
	Feedback *string `json:"feedback,omitempty"`
}

// EscalateTicketRequest es el body de POST /pqrs/{id}/escalate.
type EscalateTicketRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// CancelTicketRequest es el body de POST /pqrs/{id}/cancel.
type CancelTicketRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// TicketResponse es la representacion HTTP de un Ticket.
type TicketResponse struct {
	ID                string  `json:"id"`
	TicketYear        int32   `json:"ticket_year"`
	SerialNumber      int32   `json:"serial_number"`
	PQRType           string  `json:"pqr_type"`
	CategoryID        *string `json:"category_id,omitempty"`
	Subject           string  `json:"subject"`
	Body              string  `json:"body"`
	RequesterUserID   string  `json:"requester_user_id"`
	AssignedToUserID  *string `json:"assigned_to_user_id,omitempty"`
	AssignedAt        *string `json:"assigned_at,omitempty"`
	RespondedAt       *string `json:"responded_at,omitempty"`
	ClosedAt          *string `json:"closed_at,omitempty"`
	EscalatedAt       *string `json:"escalated_at,omitempty"`
	CancelledAt       *string `json:"cancelled_at,omitempty"`
	SLADueAt          *string `json:"sla_due_at,omitempty"`
	RequesterRating   *int32  `json:"requester_rating,omitempty"`
	RequesterFeedback *string `json:"requester_feedback,omitempty"`
	IsAnonymous       bool    `json:"is_anonymous"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	Version           int32   `json:"version"`
}

// ListTicketsResponse es el sobre del listado de tickets.
type ListTicketsResponse struct {
	Items []TicketResponse `json:"items"`
	Total int              `json:"total"`
}

// ---------------------------------------------------------------------------
// Responses
// ---------------------------------------------------------------------------

// ResponseResponse es la representacion HTTP de una Response.
type ResponseResponse struct {
	ID                string `json:"id"`
	TicketID          string `json:"ticket_id"`
	ResponseType      string `json:"response_type"`
	Body              string `json:"body"`
	RespondedByUserID string `json:"responded_by_user_id"`
	RespondedAt       string `json:"responded_at"`
	CreatedAt         string `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Status History
// ---------------------------------------------------------------------------

// StatusHistoryResponse es la representacion HTTP de un StatusHistory.
type StatusHistoryResponse struct {
	ID                   string  `json:"id"`
	TicketID             string  `json:"ticket_id"`
	FromStatus           *string `json:"from_status,omitempty"`
	ToStatus             string  `json:"to_status"`
	TransitionedByUserID string  `json:"transitioned_by_user_id"`
	TransitionedAt       string  `json:"transitioned_at"`
	Notes                *string `json:"notes,omitempty"`
}

// ListStatusHistoryResponse es el sobre del historial.
type ListStatusHistoryResponse struct {
	Items []StatusHistoryResponse `json:"items"`
	Total int                     `json:"total"`
}

// ---------------------------------------------------------------------------
// Time formatting helpers
// ---------------------------------------------------------------------------

// FormatTime formatea un time.Time como RFC3339 para JSON.
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatTimePtr formatea un *time.Time como RFC3339 string pointer.
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
