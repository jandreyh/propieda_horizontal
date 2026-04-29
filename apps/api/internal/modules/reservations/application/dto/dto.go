// Package dto contiene los Data Transfer Objects del modulo reservations.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Common Areas
// ---------------------------------------------------------------------------

// CreateCommonAreaRequest es el body de POST /common-areas.
type CreateCommonAreaRequest struct {
	Code                string   `json:"code"`
	Name                string   `json:"name"`
	Kind                string   `json:"kind"`
	MaxCapacity         *int32   `json:"max_capacity,omitempty"`
	OpeningTime         *string  `json:"opening_time,omitempty"`
	ClosingTime         *string  `json:"closing_time,omitempty"`
	SlotDurationMinutes *int32   `json:"slot_duration_minutes,omitempty"`
	CostPerUse          *float64 `json:"cost_per_use,omitempty"`
	SecurityDeposit     *float64 `json:"security_deposit,omitempty"`
	RequiresApproval    *bool    `json:"requires_approval,omitempty"`
	IsActive            *bool    `json:"is_active,omitempty"`
	Description         *string  `json:"description,omitempty"`
}

// UpdateCommonAreaRequest es el body de PUT /common-areas/{id}.
type UpdateCommonAreaRequest struct {
	Code                string   `json:"code"`
	Name                string   `json:"name"`
	Kind                string   `json:"kind"`
	MaxCapacity         *int32   `json:"max_capacity,omitempty"`
	OpeningTime         *string  `json:"opening_time,omitempty"`
	ClosingTime         *string  `json:"closing_time,omitempty"`
	SlotDurationMinutes *int32   `json:"slot_duration_minutes,omitempty"`
	CostPerUse          *float64 `json:"cost_per_use,omitempty"`
	SecurityDeposit     *float64 `json:"security_deposit,omitempty"`
	RequiresApproval    *bool    `json:"requires_approval,omitempty"`
	IsActive            *bool    `json:"is_active,omitempty"`
	Description         *string  `json:"description,omitempty"`
	Status              string   `json:"status"`
	Version             int32    `json:"version"`
}

// CommonAreaResponse es la representacion HTTP de una CommonArea.
type CommonAreaResponse struct {
	ID                  string  `json:"id"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Kind                string  `json:"kind"`
	MaxCapacity         *int32  `json:"max_capacity,omitempty"`
	OpeningTime         *string `json:"opening_time,omitempty"`
	ClosingTime         *string `json:"closing_time,omitempty"`
	SlotDurationMinutes int32   `json:"slot_duration_minutes"`
	CostPerUse          float64 `json:"cost_per_use"`
	SecurityDeposit     float64 `json:"security_deposit"`
	RequiresApproval    bool    `json:"requires_approval"`
	IsActive            bool    `json:"is_active"`
	Description         *string `json:"description,omitempty"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	Version             int32   `json:"version"`
}

// ListCommonAreasResponse es el sobre del listado de zonas comunes.
type ListCommonAreasResponse struct {
	Items []CommonAreaResponse `json:"items"`
	Total int                  `json:"total"`
}

// ---------------------------------------------------------------------------
// Blackouts
// ---------------------------------------------------------------------------

// CreateBlackoutRequest es el body de POST /common-areas/{id}/blackouts.
type CreateBlackoutRequest struct {
	FromAt string `json:"from_at"`
	ToAt   string `json:"to_at"`
	Reason string `json:"reason"`
}

// BlackoutResponse es la representacion HTTP de un ReservationBlackout.
type BlackoutResponse struct {
	ID           string `json:"id"`
	CommonAreaID string `json:"common_area_id"`
	FromAt       string `json:"from_at"`
	ToAt         string `json:"to_at"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	Version      int32  `json:"version"`
}

// ---------------------------------------------------------------------------
// Reservations
// ---------------------------------------------------------------------------

// CreateReservationRequest es el body de POST /reservations.
type CreateReservationRequest struct {
	CommonAreaID   string  `json:"common_area_id"`
	UnitID         string  `json:"unit_id"`
	SlotStartAt    string  `json:"slot_start_at"`
	SlotEndAt      string  `json:"slot_end_at"`
	AttendeesCount *int32  `json:"attendees_count,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	IdempotencyKey *string `json:"idempotency_key,omitempty"`
}

// ReservationResponse es la representacion HTTP de una Reservation.
type ReservationResponse struct {
	ID                string  `json:"id"`
	CommonAreaID      string  `json:"common_area_id"`
	UnitID            string  `json:"unit_id"`
	RequestedByUserID string  `json:"requested_by_user_id"`
	SlotStartAt       string  `json:"slot_start_at"`
	SlotEndAt         string  `json:"slot_end_at"`
	AttendeesCount    *int32  `json:"attendees_count,omitempty"`
	Cost              float64 `json:"cost"`
	SecurityDeposit   float64 `json:"security_deposit"`
	DepositRefunded   bool    `json:"deposit_refunded"`
	QRCodeHash        *string `json:"qr_code_hash,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	ApprovedBy        *string `json:"approved_by,omitempty"`
	ApprovedAt        *string `json:"approved_at,omitempty"`
	CancelledBy       *string `json:"cancelled_by,omitempty"`
	CancelledAt       *string `json:"cancelled_at,omitempty"`
	ConsumedAt        *string `json:"consumed_at,omitempty"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	Version           int32   `json:"version"`
}

// ListReservationsResponse es el sobre del listado de reservas.
type ListReservationsResponse struct {
	Items []ReservationResponse `json:"items"`
	Total int                   `json:"total"`
}

// ---------------------------------------------------------------------------
// Availability
// ---------------------------------------------------------------------------

// AvailabilitySlot representa un slot disponible en la respuesta de
// disponibilidad.
type AvailabilitySlot struct {
	SlotStart   string `json:"slot_start"`
	SlotEnd     string `json:"slot_end"`
	IsAvailable bool   `json:"is_available"`
}

// AvailabilityResponse es la respuesta de GET /common-areas/{id}/availability.
type AvailabilityResponse struct {
	CommonAreaID string             `json:"common_area_id"`
	Date         string             `json:"date"`
	Slots        []AvailabilitySlot `json:"slots"`
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
