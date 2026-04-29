// Package dto contiene los Data Transfer Objects del modulo penalties.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Penalty Catalog
// ---------------------------------------------------------------------------

// CreateCatalogRequest es el body de POST /penalty-catalog.
type CreateCatalogRequest struct {
	Code                     string   `json:"code"`
	Name                     string   `json:"name"`
	Description              *string  `json:"description,omitempty"`
	DefaultSanctionType      string   `json:"default_sanction_type"`
	BaseAmount               float64  `json:"base_amount"`
	RecurrenceMultiplier     float64  `json:"recurrence_multiplier"`
	RecurrenceCAPMultiplier  float64  `json:"recurrence_cap_multiplier"`
	RequiresCouncilThreshold *float64 `json:"requires_council_threshold,omitempty"`
}

// UpdateCatalogRequest es el body de PATCH /penalty-catalog/{id}.
type UpdateCatalogRequest struct {
	Code                     string   `json:"code"`
	Name                     string   `json:"name"`
	Description              *string  `json:"description,omitempty"`
	DefaultSanctionType      string   `json:"default_sanction_type"`
	BaseAmount               float64  `json:"base_amount"`
	RecurrenceMultiplier     float64  `json:"recurrence_multiplier"`
	RecurrenceCAPMultiplier  float64  `json:"recurrence_cap_multiplier"`
	RequiresCouncilThreshold *float64 `json:"requires_council_threshold,omitempty"`
	Status                   string   `json:"status"`
	Version                  int32    `json:"version"`
}

// CatalogResponse es la representacion HTTP de un PenaltyCatalog.
type CatalogResponse struct {
	ID                       string   `json:"id"`
	Code                     string   `json:"code"`
	Name                     string   `json:"name"`
	Description              *string  `json:"description,omitempty"`
	DefaultSanctionType      string   `json:"default_sanction_type"`
	BaseAmount               float64  `json:"base_amount"`
	RecurrenceMultiplier     float64  `json:"recurrence_multiplier"`
	RecurrenceCAPMultiplier  float64  `json:"recurrence_cap_multiplier"`
	RequiresCouncilThreshold *float64 `json:"requires_council_threshold,omitempty"`
	Status                   string   `json:"status"`
	CreatedAt                string   `json:"created_at"`
	UpdatedAt                string   `json:"updated_at"`
	Version                  int32    `json:"version"`
}

// ListCatalogResponse es el sobre del listado del catalogo.
type ListCatalogResponse struct {
	Items []CatalogResponse `json:"items"`
	Total int               `json:"total"`
}

// ---------------------------------------------------------------------------
// Penalties
// ---------------------------------------------------------------------------

// ImposePenaltyRequest es el body de POST /penalties.
type ImposePenaltyRequest struct {
	CatalogID        string  `json:"catalog_id"`
	DebtorUserID     string  `json:"debtor_user_id"`
	UnitID           *string `json:"unit_id,omitempty"`
	SourceIncidentID *string `json:"source_incident_id,omitempty"`
	SanctionType     *string `json:"sanction_type,omitempty"`
	Reason           string  `json:"reason"`
	IdempotencyKey   *string `json:"idempotency_key,omitempty"`
}

// PenaltyResponse es la representacion HTTP de un Penalty.
type PenaltyResponse struct {
	ID                      string  `json:"id"`
	CatalogID               string  `json:"catalog_id"`
	DebtorUserID            string  `json:"debtor_user_id"`
	UnitID                  *string `json:"unit_id,omitempty"`
	SourceIncidentID        *string `json:"source_incident_id,omitempty"`
	SanctionType            string  `json:"sanction_type"`
	Amount                  float64 `json:"amount"`
	Reason                  string  `json:"reason"`
	ImposedByUserID         string  `json:"imposed_by_user_id"`
	NotifiedAt              *string `json:"notified_at,omitempty"`
	AppealDeadlineAt        *string `json:"appeal_deadline_at,omitempty"`
	ConfirmedAt             *string `json:"confirmed_at,omitempty"`
	SettledAt               *string `json:"settled_at,omitempty"`
	DismissedAt             *string `json:"dismissed_at,omitempty"`
	CancelledAt             *string `json:"cancelled_at,omitempty"`
	RequiresCouncilApproval bool    `json:"requires_council_approval"`
	CouncilApprovedByUserID *string `json:"council_approved_by_user_id,omitempty"`
	CouncilApprovedAt       *string `json:"council_approved_at,omitempty"`
	Status                  string  `json:"status"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
	Version                 int32   `json:"version"`
}

// ListPenaltiesResponse es el sobre del listado de sanciones.
type ListPenaltiesResponse struct {
	Items []PenaltyResponse `json:"items"`
	Total int               `json:"total"`
}

// ---------------------------------------------------------------------------
// Appeals
// ---------------------------------------------------------------------------

// SubmitAppealRequest es el body de POST /penalties/{id}/appeals.
type SubmitAppealRequest struct {
	Grounds string `json:"grounds"`
}

// ResolveAppealRequest es el body de POST /penalties/{id}/appeals/{aid}/resolve.
type ResolveAppealRequest struct {
	Resolution string `json:"resolution"`
	Status     string `json:"status"`
	Version    int32  `json:"version"`
}

// AppealResponse es la representacion HTTP de un PenaltyAppeal.
type AppealResponse struct {
	ID                string  `json:"id"`
	PenaltyID         string  `json:"penalty_id"`
	SubmittedByUserID string  `json:"submitted_by_user_id"`
	SubmittedAt       string  `json:"submitted_at"`
	Grounds           string  `json:"grounds"`
	ResolvedByUserID  *string `json:"resolved_by_user_id,omitempty"`
	ResolvedAt        *string `json:"resolved_at,omitempty"`
	Resolution        *string `json:"resolution,omitempty"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	Version           int32   `json:"version"`
}

// ---------------------------------------------------------------------------
// Status History
// ---------------------------------------------------------------------------

// StatusHistoryResponse es la representacion HTTP de un registro de
// historial de transiciones.
type StatusHistoryResponse struct {
	ID                   string  `json:"id"`
	PenaltyID            string  `json:"penalty_id"`
	FromStatus           *string `json:"from_status,omitempty"`
	ToStatus             string  `json:"to_status"`
	TransitionedByUserID string  `json:"transitioned_by_user_id"`
	TransitionedAt       string  `json:"transitioned_at"`
	Notes                *string `json:"notes,omitempty"`
}

// ListStatusHistoryResponse es el sobre del listado de historial.
type ListStatusHistoryResponse struct {
	Items []StatusHistoryResponse `json:"items"`
	Total int                     `json:"total"`
}

// ---------------------------------------------------------------------------
// Time formatting helper
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
