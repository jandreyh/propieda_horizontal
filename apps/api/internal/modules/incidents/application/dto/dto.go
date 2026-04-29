// Package dto contiene los Data Transfer Objects del modulo incidents.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Incidents
// ---------------------------------------------------------------------------

// ReportIncidentRequest es el body de POST /incidents.
type ReportIncidentRequest struct {
	IncidentType   string  `json:"incident_type"`
	Severity       string  `json:"severity"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	StructureID    *string `json:"structure_id,omitempty"`
	LocationDetail *string `json:"location_detail,omitempty"`
}

// IncidentResponse es la representacion HTTP de un Incident.
type IncidentResponse struct {
	ID               string  `json:"id"`
	IncidentType     string  `json:"incident_type"`
	Severity         string  `json:"severity"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	ReportedByUserID string  `json:"reported_by_user_id"`
	ReportedAt       string  `json:"reported_at"`
	StructureID      *string `json:"structure_id,omitempty"`
	LocationDetail   *string `json:"location_detail,omitempty"`
	AssignedToUserID *string `json:"assigned_to_user_id,omitempty"`
	AssignedAt       *string `json:"assigned_at,omitempty"`
	StartedAt        *string `json:"started_at,omitempty"`
	ResolvedAt       *string `json:"resolved_at,omitempty"`
	ClosedAt         *string `json:"closed_at,omitempty"`
	CancelledAt      *string `json:"cancelled_at,omitempty"`
	ResolutionNotes  *string `json:"resolution_notes,omitempty"`
	Escalated        bool    `json:"escalated"`
	SLAAssignDueAt   *string `json:"sla_assign_due_at,omitempty"`
	SLAResolveDueAt  *string `json:"sla_resolve_due_at,omitempty"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	Version          int32   `json:"version"`
}

// ListIncidentsResponse es el sobre del listado de incidentes.
type ListIncidentsResponse struct {
	Items []IncidentResponse `json:"items"`
	Total int                `json:"total"`
}

// ---------------------------------------------------------------------------
// Assign
// ---------------------------------------------------------------------------

// AssignIncidentRequest es el body de POST /incidents/{id}/assign.
type AssignIncidentRequest struct {
	AssignedToUserID string `json:"assigned_to_user_id"`
}

// ---------------------------------------------------------------------------
// Resolve / Close
// ---------------------------------------------------------------------------

// ResolveIncidentRequest es el body de POST /incidents/{id}/resolve.
type ResolveIncidentRequest struct {
	ResolutionNotes string `json:"resolution_notes"`
}

// CloseIncidentRequest es el body de POST /incidents/{id}/close.
type CloseIncidentRequest struct {
	ResolutionNotes string `json:"resolution_notes"`
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

// AddAttachmentRequest es el body de POST /incidents/{id}/attachments.
type AddAttachmentRequest struct {
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// AttachmentResponse es la representacion HTTP de un Attachment.
type AttachmentResponse struct {
	ID         string `json:"id"`
	IncidentID string `json:"incident_id"`
	URL        string `json:"url"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
	UploadedBy string `json:"uploaded_by"`
	CreatedAt  string `json:"created_at"`
}

// ListAttachmentsResponse es el sobre del listado de adjuntos.
type ListAttachmentsResponse struct {
	Items []AttachmentResponse `json:"items"`
	Total int                  `json:"total"`
}

// ---------------------------------------------------------------------------
// Status History
// ---------------------------------------------------------------------------

// StatusHistoryResponse es la representacion HTTP de un StatusHistory.
type StatusHistoryResponse struct {
	ID                   string  `json:"id"`
	IncidentID           string  `json:"incident_id"`
	FromStatus           *string `json:"from_status,omitempty"`
	ToStatus             string  `json:"to_status"`
	TransitionedByUserID string  `json:"transitioned_by_user_id"`
	TransitionedAt       string  `json:"transitioned_at"`
	Notes                *string `json:"notes,omitempty"`
}

// ListStatusHistoryResponse es el sobre del historial de estados.
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
