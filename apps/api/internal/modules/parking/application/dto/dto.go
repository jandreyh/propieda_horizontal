// Package dto contiene los Data Transfer Objects del modulo parking.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Parking Spaces
// ---------------------------------------------------------------------------

// CreateSpaceRequest es el body de POST /parking-spaces.
type CreateSpaceRequest struct {
	Code        string   `json:"code"`
	Type        string   `json:"type"`
	StructureID *string  `json:"structure_id,omitempty"`
	Level       *string  `json:"level,omitempty"`
	Zone        *string  `json:"zone,omitempty"`
	MonthlyFee  *float64 `json:"monthly_fee,omitempty"`
	IsVisitor   bool     `json:"is_visitor"`
	Notes       *string  `json:"notes,omitempty"`
}

// UpdateSpaceRequest es el body de PUT /parking-spaces/{id}.
type UpdateSpaceRequest struct {
	Code        string   `json:"code"`
	Type        string   `json:"type"`
	StructureID *string  `json:"structure_id,omitempty"`
	Level       *string  `json:"level,omitempty"`
	Zone        *string  `json:"zone,omitempty"`
	MonthlyFee  *float64 `json:"monthly_fee,omitempty"`
	IsVisitor   bool     `json:"is_visitor"`
	Notes       *string  `json:"notes,omitempty"`
	Status      string   `json:"status"`
	Version     int32    `json:"version"`
}

// SpaceResponse es la representacion HTTP de un ParkingSpace.
type SpaceResponse struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Type        string   `json:"type"`
	StructureID *string  `json:"structure_id,omitempty"`
	Level       *string  `json:"level,omitempty"`
	Zone        *string  `json:"zone,omitempty"`
	MonthlyFee  *float64 `json:"monthly_fee,omitempty"`
	IsVisitor   bool     `json:"is_visitor"`
	Notes       *string  `json:"notes,omitempty"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Version     int32    `json:"version"`
}

// ListSpacesResponse es el sobre del listado de espacios.
type ListSpacesResponse struct {
	Items []SpaceResponse `json:"items"`
	Total int             `json:"total"`
}

// ---------------------------------------------------------------------------
// Parking Assignments
// ---------------------------------------------------------------------------

// AssignSpaceRequest es el body de POST /parking-spaces/{id}/assign.
type AssignSpaceRequest struct {
	UnitID    string  `json:"unit_id"`
	VehicleID *string `json:"vehicle_id,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

// ReleaseAssignmentRequest es el body de POST /parking-assignments/{id}/release.
type ReleaseAssignmentRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// AssignmentResponse es la representacion HTTP de un ParkingAssignment.
type AssignmentResponse struct {
	ID               string  `json:"id"`
	ParkingSpaceID   string  `json:"parking_space_id"`
	UnitID           string  `json:"unit_id"`
	VehicleID        *string `json:"vehicle_id,omitempty"`
	AssignedByUserID *string `json:"assigned_by_user_id,omitempty"`
	SinceDate        string  `json:"since_date"`
	UntilDate        *string `json:"until_date,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	Version          int32   `json:"version"`
}

// UnitParkingResponse es la respuesta de GET /units/{id}/parking.
type UnitParkingResponse struct {
	Assignments  []AssignmentResponse  `json:"assignments"`
	Reservations []ReservationResponse `json:"reservations"`
}

// ---------------------------------------------------------------------------
// Visitor Reservations
// ---------------------------------------------------------------------------

// CreateVisitorReservationRequest es el body de POST /parking-visitor-reservations.
type CreateVisitorReservationRequest struct {
	ParkingSpaceID  string  `json:"parking_space_id"`
	UnitID          string  `json:"unit_id"`
	VisitorName     string  `json:"visitor_name"`
	VisitorDocument *string `json:"visitor_document,omitempty"`
	VehiclePlate    *string `json:"vehicle_plate,omitempty"`
	SlotStartAt     string  `json:"slot_start_at"`
	SlotEndAt       string  `json:"slot_end_at"`
	IdempotencyKey  *string `json:"idempotency_key,omitempty"`
}

// ReservationResponse es la representacion HTTP de una VisitorReservation.
type ReservationResponse struct {
	ID              string  `json:"id"`
	ParkingSpaceID  string  `json:"parking_space_id"`
	UnitID          string  `json:"unit_id"`
	RequestedBy     string  `json:"requested_by"`
	VisitorName     string  `json:"visitor_name"`
	VisitorDocument *string `json:"visitor_document,omitempty"`
	VehiclePlate    *string `json:"vehicle_plate,omitempty"`
	SlotStartAt     string  `json:"slot_start_at"`
	SlotEndAt       string  `json:"slot_end_at"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	Version         int32   `json:"version"`
}

// ListReservationsResponse es el sobre del listado de reservas.
type ListReservationsResponse struct {
	Items []ReservationResponse `json:"items"`
	Total int                   `json:"total"`
}

// ---------------------------------------------------------------------------
// Lotteries
// ---------------------------------------------------------------------------

// RunLotteryRequest es el body de POST /parking-lotteries/run.
type RunLotteryRequest struct {
	Name          string   `json:"name"`
	Seed          string   `json:"seed"`
	SpaceIDs      []string `json:"space_ids"`
	EligibleUnits []string `json:"eligible_units"`
}

// LotteryRunResponse es la representacion HTTP de un LotteryRun.
type LotteryRunResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SeedHash   string `json:"seed_hash"`
	ExecutedAt string `json:"executed_at"`
	ExecutedBy string `json:"executed_by"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Version    int32  `json:"version"`
}

// LotteryResultResponse es la representacion HTTP de un LotteryResult.
type LotteryResultResponse struct {
	ID             string  `json:"id"`
	LotteryRunID   string  `json:"lottery_run_id"`
	UnitID         string  `json:"unit_id"`
	ParkingSpaceID *string `json:"parking_space_id,omitempty"`
	Position       int32   `json:"position"`
	Status         string  `json:"status"`
}

// LotteryResultsResponse es la respuesta de GET /parking-lotteries/{id}/results.
type LotteryResultsResponse struct {
	Run     LotteryRunResponse      `json:"run"`
	Results []LotteryResultResponse `json:"results"`
}

// ---------------------------------------------------------------------------
// Guard View
// ---------------------------------------------------------------------------

// GuardParkingEntryResponse es una entrada individual en la vista del
// guarda (solo campos necesarios para verificacion, sin datos sensibles).
type GuardParkingEntryResponse struct {
	SpaceCode    string  `json:"space_code"`
	SpaceType    string  `json:"space_type"`
	UnitID       *string `json:"unit_id,omitempty"`
	VehiclePlate *string `json:"vehicle_plate,omitempty"`
	VisitorName  *string `json:"visitor_name,omitempty"`
	SlotStartAt  *string `json:"slot_start_at,omitempty"`
	SlotEndAt    *string `json:"slot_end_at,omitempty"`
	EntryType    string  `json:"entry_type"`
}

// GuardParkingTodayResponse es la respuesta de GET /guard/parking/today.
type GuardParkingTodayResponse struct {
	Date    string                      `json:"date"`
	Entries []GuardParkingEntryResponse `json:"entries"`
	Total   int                         `json:"total"`
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
