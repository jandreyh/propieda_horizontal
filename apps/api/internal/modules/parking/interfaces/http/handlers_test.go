package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/parking/domain"
	"github.com/saas-ph/api/internal/modules/parking/domain/entities"
	parkinghttp "github.com/saas-ph/api/internal/modules/parking/interfaces/http"
)

// --- in-memory fakes (minimo necesario para los tests) ---

type stubSpaces struct {
	spaces []entities.ParkingSpace
}

func (s *stubSpaces) Create(_ context.Context, in domain.CreateSpaceInput) (entities.ParkingSpace, error) {
	for _, sp := range s.spaces {
		if sp.Code == in.Code {
			return entities.ParkingSpace{}, domain.ErrSpaceCodeDuplicate
		}
	}
	now := time.Now()
	space := entities.ParkingSpace{
		ID:        "11111111-1111-1111-1111-111111111111",
		Code:      in.Code,
		Type:      in.Type,
		IsVisitor: in.IsVisitor,
		Status:    entities.SpaceStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
	s.spaces = append(s.spaces, space)
	return space, nil
}

func (s *stubSpaces) GetByID(_ context.Context, id string) (entities.ParkingSpace, error) {
	for _, sp := range s.spaces {
		if sp.ID == id {
			return sp, nil
		}
	}
	return entities.ParkingSpace{}, domain.ErrSpaceNotFound
}

func (s *stubSpaces) GetByCode(_ context.Context, code string) (entities.ParkingSpace, error) {
	for _, sp := range s.spaces {
		if sp.Code == code {
			return sp, nil
		}
	}
	return entities.ParkingSpace{}, domain.ErrSpaceNotFound
}

func (s *stubSpaces) List(_ context.Context) ([]entities.ParkingSpace, error) {
	return s.spaces, nil
}

func (s *stubSpaces) Update(_ context.Context, in domain.UpdateSpaceInput) (entities.ParkingSpace, error) {
	for i, sp := range s.spaces {
		if sp.ID == in.ID {
			if sp.Version != in.ExpectedVersion {
				return entities.ParkingSpace{}, domain.ErrVersionConflict
			}
			s.spaces[i].Code = in.Code
			s.spaces[i].Type = in.Type
			s.spaces[i].Status = in.Status
			s.spaces[i].Version++
			return s.spaces[i], nil
		}
	}
	return entities.ParkingSpace{}, domain.ErrSpaceNotFound
}

func (s *stubSpaces) SoftDelete(_ context.Context, _ string, _ int32, _ string) error {
	return nil
}

type stubAssignments struct {
	assignments []entities.ParkingAssignment
}

func (s *stubAssignments) Create(_ context.Context, in domain.CreateAssignmentInput) (entities.ParkingAssignment, error) {
	// Check for active assignment on the same space.
	for _, a := range s.assignments {
		if a.ParkingSpaceID == in.ParkingSpaceID && a.UntilDate == nil && a.Status == entities.AssignmentStatusActive {
			return entities.ParkingAssignment{}, domain.ErrAssignmentAlreadyActive
		}
	}
	now := time.Now()
	assignment := entities.ParkingAssignment{
		ID:               "22222222-2222-2222-2222-222222222222",
		ParkingSpaceID:   in.ParkingSpaceID,
		UnitID:           in.UnitID,
		VehicleID:        in.VehicleID,
		AssignedByUserID: in.AssignedByUserID,
		SinceDate:        in.SinceDate,
		Status:           entities.AssignmentStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		Version:          1,
	}
	s.assignments = append(s.assignments, assignment)
	return assignment, nil
}

func (s *stubAssignments) GetByID(_ context.Context, id string) (entities.ParkingAssignment, error) {
	for _, a := range s.assignments {
		if a.ID == id {
			return a, nil
		}
	}
	return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
}

func (s *stubAssignments) GetActiveBySpaceID(_ context.Context, spaceID string) (entities.ParkingAssignment, error) {
	for _, a := range s.assignments {
		if a.ParkingSpaceID == spaceID && a.UntilDate == nil && a.Status == entities.AssignmentStatusActive {
			return a, nil
		}
	}
	return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
}

func (s *stubAssignments) ListActiveByUnitID(_ context.Context, unitID string) ([]entities.ParkingAssignment, error) {
	var result []entities.ParkingAssignment
	for _, a := range s.assignments {
		if a.UnitID == unitID && a.UntilDate == nil && a.Status == entities.AssignmentStatusActive {
			result = append(result, a)
		}
	}
	return result, nil
}

func (s *stubAssignments) ListBySpaceID(_ context.Context, _ string) ([]entities.ParkingAssignment, error) {
	return nil, nil
}

func (s *stubAssignments) CloseAssignment(_ context.Context, id string, untilDate time.Time, expectedVersion int32, _ string) (entities.ParkingAssignment, error) {
	for i, a := range s.assignments {
		if a.ID == id {
			if a.Version != expectedVersion {
				return entities.ParkingAssignment{}, domain.ErrVersionConflict
			}
			s.assignments[i].UntilDate = &untilDate
			s.assignments[i].Status = entities.AssignmentStatusClosed
			s.assignments[i].Version++
			return s.assignments[i], nil
		}
	}
	return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
}

func (s *stubAssignments) SoftDelete(_ context.Context, _ string, _ int32, _ string) error {
	return nil
}

type stubHistory struct{}

func (s *stubHistory) Record(_ context.Context, _ domain.RecordHistoryInput) (entities.AssignmentHistory, error) {
	return entities.AssignmentHistory{}, nil
}
func (s *stubHistory) ListBySpaceID(_ context.Context, _ string) ([]entities.AssignmentHistory, error) {
	return nil, nil
}
func (s *stubHistory) ListByUnitID(_ context.Context, _ string) ([]entities.AssignmentHistory, error) {
	return nil, nil
}

type stubReservations struct {
	reservations []entities.VisitorReservation
}

func (s *stubReservations) Create(_ context.Context, in domain.CreateReservationInput) (entities.VisitorReservation, error) {
	// Check for slot conflicts.
	for _, r := range s.reservations {
		if r.ParkingSpaceID == in.ParkingSpaceID &&
			r.Status == entities.ReservationStatusConfirmed &&
			r.SlotStartAt.Equal(in.SlotStartAt) {
			return entities.VisitorReservation{}, domain.ErrReservationSlotConflict
		}
	}
	now := time.Now()
	reservation := entities.VisitorReservation{
		ID:              "33333333-3333-3333-3333-333333333333",
		ParkingSpaceID:  in.ParkingSpaceID,
		UnitID:          in.UnitID,
		RequestedBy:     in.RequestedBy,
		VisitorName:     in.VisitorName,
		VisitorDocument: in.VisitorDocument,
		VehiclePlate:    in.VehiclePlate,
		SlotStartAt:     in.SlotStartAt,
		SlotEndAt:       in.SlotEndAt,
		IdempotencyKey:  in.IdempotencyKey,
		Status:          entities.ReservationStatusConfirmed,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}
	s.reservations = append(s.reservations, reservation)
	return reservation, nil
}

func (s *stubReservations) GetByID(_ context.Context, id string) (entities.VisitorReservation, error) {
	for _, r := range s.reservations {
		if r.ID == id {
			return r, nil
		}
	}
	return entities.VisitorReservation{}, domain.ErrReservationNotFound
}

func (s *stubReservations) ListByDate(_ context.Context, _, _ time.Time) ([]entities.VisitorReservation, error) {
	return s.reservations, nil
}

func (s *stubReservations) ListByUnit(_ context.Context, unitID string) ([]entities.VisitorReservation, error) {
	var result []entities.VisitorReservation
	for _, r := range s.reservations {
		if r.UnitID == unitID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (s *stubReservations) UpdateStatus(_ context.Context, id string, expectedVersion int32, newStatus entities.ReservationStatus, _ string) (entities.VisitorReservation, error) {
	for i, r := range s.reservations {
		if r.ID == id {
			if r.Version != expectedVersion {
				return entities.VisitorReservation{}, domain.ErrVersionConflict
			}
			s.reservations[i].Status = newStatus
			s.reservations[i].Version++
			return s.reservations[i], nil
		}
	}
	return entities.VisitorReservation{}, domain.ErrReservationNotFound
}

func (s *stubReservations) Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.VisitorReservation, error) {
	return s.UpdateStatus(ctx, id, expectedVersion, entities.ReservationStatusCancelled, actorID)
}

func (s *stubReservations) GetByIdempotencyKey(_ context.Context, _ string) (entities.VisitorReservation, error) {
	return entities.VisitorReservation{}, domain.ErrReservationNotFound
}

type stubLotteryRuns struct{}

func (s *stubLotteryRuns) Create(_ context.Context, _ domain.CreateLotteryRunInput) (entities.LotteryRun, error) {
	return entities.LotteryRun{}, nil
}
func (s *stubLotteryRuns) GetByID(_ context.Context, _ string) (entities.LotteryRun, error) {
	return entities.LotteryRun{}, domain.ErrLotteryNotFound
}
func (s *stubLotteryRuns) List(_ context.Context) ([]entities.LotteryRun, error) {
	return nil, nil
}

type stubLotteryResults struct{}

func (s *stubLotteryResults) CreateBatch(_ context.Context, _ []domain.CreateLotteryResultInput) ([]entities.LotteryResult, error) {
	return nil, nil
}
func (s *stubLotteryResults) ListByRunID(_ context.Context, _ string) ([]entities.LotteryResult, error) {
	return nil, nil
}

type stubOutbox struct{}

func (s *stubOutbox) Enqueue(_ context.Context, _ domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	return entities.OutboxEvent{}, nil
}
func (s *stubOutbox) LockPending(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
	return nil, nil
}
func (s *stubOutbox) MarkDelivered(_ context.Context, _ string) error { return nil }
func (s *stubOutbox) MarkFailed(_ context.Context, _, _ string, _ int) error {
	return nil
}

// conflictingAssignments simulates a race condition where Create always
// fails with ErrAssignmentAlreadyActive (the DB UNIQUE constraint
// rejects the INSERT).
type conflictingAssignments struct{}

func (s *conflictingAssignments) Create(_ context.Context, _ domain.CreateAssignmentInput) (entities.ParkingAssignment, error) {
	return entities.ParkingAssignment{}, domain.ErrAssignmentAlreadyActive
}
func (s *conflictingAssignments) GetByID(_ context.Context, _ string) (entities.ParkingAssignment, error) {
	return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
}
func (s *conflictingAssignments) GetActiveBySpaceID(_ context.Context, _ string) (entities.ParkingAssignment, error) {
	return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
}
func (s *conflictingAssignments) ListActiveByUnitID(_ context.Context, _ string) ([]entities.ParkingAssignment, error) {
	return nil, nil
}
func (s *conflictingAssignments) ListBySpaceID(_ context.Context, _ string) ([]entities.ParkingAssignment, error) {
	return nil, nil
}
func (s *conflictingAssignments) CloseAssignment(_ context.Context, _ string, _ time.Time, _ int32, _ string) (entities.ParkingAssignment, error) {
	return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
}
func (s *conflictingAssignments) SoftDelete(_ context.Context, _ string, _ int32, _ string) error {
	return nil
}

// visitorSpace is a pre-seeded visitor space for reservation tests.
var visitorSpace = entities.ParkingSpace{
	ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	Code:      "V-01",
	Type:      entities.SpaceTypeVisitor,
	IsVisitor: true,
	Status:    entities.SpaceStatusActive,
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
	Version:   1,
}

// privateSpace is a pre-seeded private space for assignment tests.
var privateSpace = entities.ParkingSpace{
	ID:        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	Code:      "P-01",
	Type:      entities.SpaceTypeCovered,
	IsVisitor: false,
	Status:    entities.SpaceStatusActive,
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
	Version:   1,
}

func mountTest(t *testing.T) (*chi.Mux, *stubSpaces, *stubAssignments, *stubReservations) {
	t.Helper()
	spaces := &stubSpaces{spaces: []entities.ParkingSpace{visitorSpace, privateSpace}}
	assignments := &stubAssignments{}
	reservations := &stubReservations{}
	r := chi.NewRouter()
	parkinghttp.Mount(r, parkinghttp.Dependencies{
		Spaces:       spaces,
		Assignments:  assignments,
		History:      &stubHistory{},
		Reservations: reservations,
		Lotteries:    &stubLotteryRuns{},
		Results:      &stubLotteryResults{},
		Outbox:       &stubOutbox{},
	})
	return r, spaces, assignments, reservations
}

// TestCreateSpace_Success verifica que POST /parking-spaces con datos
// validos devuelve 201 con el espacio creado.
func TestCreateSpace_Success(t *testing.T) {
	r, _, _, _ := mountTest(t)
	body := []byte(`{
		"code": "A-101",
		"type": "covered",
		"is_visitor": false
	}`)
	req := httptest.NewRequest(http.MethodPost, "/parking-spaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["code"] != "A-101" {
		t.Errorf("expected code A-101, got %v", resp["code"])
	}
	if resp["status"] != "active" {
		t.Errorf("expected status active, got %v", resp["status"])
	}
}

// TestCreateSpace_InvalidCode verifica que POST /parking-spaces con un
// codigo invalido (vacio) devuelve 400 + Problem JSON.
func TestCreateSpace_InvalidCode(t *testing.T) {
	r, _, _, _ := mountTest(t)
	body := []byte(`{
		"code": "",
		"type": "covered",
		"is_visitor": false
	}`)
	req := httptest.NewRequest(http.MethodPost, "/parking-spaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("expected problem+json, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "code") {
		t.Errorf("expected body to mention 'code', got %s", rec.Body.String())
	}
}

// TestAssignSpace_Conflict verifica que asignar un espacio que ya tiene
// una asignacion activa (simulando la constraint UNIQUE de la DB que
// falla en un race condition) devuelve 409 + Problem JSON.
func TestAssignSpace_Conflict(t *testing.T) {
	// Use a special assignments stub that always returns
	// ErrAssignmentAlreadyActive from Create, simulating a race
	// condition where both requests pass GetActiveBySpaceID but the
	// DB UNIQUE constraint rejects the second INSERT.
	spaces := &stubSpaces{spaces: []entities.ParkingSpace{privateSpace}}
	conflicting := &conflictingAssignments{}
	r := chi.NewRouter()
	parkinghttp.Mount(r, parkinghttp.Dependencies{
		Spaces:       spaces,
		Assignments:  conflicting,
		History:      &stubHistory{},
		Reservations: &stubReservations{},
		Lotteries:    &stubLotteryRuns{},
		Results:      &stubLotteryResults{},
		Outbox:       &stubOutbox{},
	})

	body := []byte(`{
		"unit_id": "44444444-4444-4444-4444-444444444444"
	}`)
	url := "/parking-spaces/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/assign"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = parkinghttp.WithActorID(req, "55555555-5555-5555-5555-555555555555")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("expected problem+json, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "active assignment") {
		t.Errorf("expected body to mention 'active assignment', got %s", rec.Body.String())
	}
}

// TestCreateVisitorReservation_SlotConflict verifica que intentar
// reservar el mismo slot dos veces devuelve 409 en el segundo intento.
func TestCreateVisitorReservation_SlotConflict(t *testing.T) {
	r, _, _, _ := mountTest(t)
	slotStart := time.Now().Add(2 * time.Hour).Truncate(time.Second).UTC().Format(time.RFC3339)
	slotEnd := time.Now().Add(4 * time.Hour).Truncate(time.Second).UTC().Format(time.RFC3339)
	bodyJSON := `{
		"parking_space_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"unit_id": "44444444-4444-4444-4444-444444444444",
		"visitor_name": "Juan Perez",
		"slot_start_at": "` + slotStart + `",
		"slot_end_at": "` + slotEnd + `"
	}`

	url := "/parking-visitor-reservations"

	// First reservation - should succeed.
	req1 := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(bodyJSON)))
	req1.Header.Set("Content-Type", "application/json")
	req1 = parkinghttp.WithActorID(req1, "55555555-5555-5555-5555-555555555555")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first reservation: expected 201, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Second reservation same slot - should be 409.
	req2 := httptest.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(bodyJSON)))
	req2.Header.Set("Content-Type", "application/json")
	req2 = parkinghttp.WithActorID(req2, "55555555-5555-5555-5555-555555555555")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("second reservation: expected 409, got %d: %s", rec2.Code, rec2.Body.String())
	}
	ct := rec2.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("expected problem+json, got %q", ct)
	}
}

// TestGuardView_LimitedFields verifica que GET /guard/parking/today
// devuelve la vista del guarda con campos limitados (no datos sensibles)
// y con la estructura esperada.
func TestGuardView_LimitedFields(t *testing.T) {
	r, _, assignments, reservations := mountTest(t)

	// Seed an active assignment.
	now := time.Now()
	assignments.assignments = append(assignments.assignments, entities.ParkingAssignment{
		ID:             "77777777-7777-7777-7777-777777777777",
		ParkingSpaceID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UnitID:         "44444444-4444-4444-4444-444444444444",
		SinceDate:      now.Add(-24 * time.Hour),
		Status:         entities.AssignmentStatusActive,
		Version:        1,
	})

	// Seed a confirmed visitor reservation for today.
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(2 * time.Hour)
	plate := "ABC-123"
	reservations.reservations = append(reservations.reservations, entities.VisitorReservation{
		ID:             "88888888-8888-8888-8888-888888888888",
		ParkingSpaceID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UnitID:         "44444444-4444-4444-4444-444444444444",
		RequestedBy:    "55555555-5555-5555-5555-555555555555",
		VisitorName:    "Maria Garcia",
		VehiclePlate:   &plate,
		SlotStartAt:    startOfDay,
		SlotEndAt:      endOfDay,
		Status:         entities.ReservationStatusConfirmed,
		Version:        1,
	})

	req := httptest.NewRequest(http.MethodGet, "/guard/parking/today", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Date    string `json:"date"`
		Entries []struct {
			SpaceCode    string  `json:"space_code"`
			SpaceType    string  `json:"space_type"`
			UnitID       *string `json:"unit_id"`
			VehiclePlate *string `json:"vehicle_plate"`
			VisitorName  *string `json:"visitor_name"`
			EntryType    string  `json:"entry_type"`
		} `json:"entries"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if resp.Date == "" {
		t.Error("expected date to be set")
	}
	if resp.Total < 2 {
		t.Errorf("expected at least 2 entries, got %d", resp.Total)
	}

	// Verify the response does NOT contain sensitive fields like
	// monthly_fee, created_by, etc. - only guard-relevant fields.
	bodyStr := rec.Body.String()
	sensitiveFields := []string{"monthly_fee", "created_by", "deleted_at", "document_number"}
	for _, field := range sensitiveFields {
		if strings.Contains(bodyStr, field) {
			t.Errorf("guard view should NOT contain sensitive field %q", field)
		}
	}

	// Verify we have both entry types.
	hasAssignment := false
	hasVisitor := false
	for _, e := range resp.Entries {
		if e.EntryType == "assignment" {
			hasAssignment = true
			if e.SpaceCode == "" {
				t.Error("assignment entry should have space_code")
			}
		}
		if e.EntryType == "visitor" {
			hasVisitor = true
			if e.VisitorName == nil || *e.VisitorName == "" {
				t.Error("visitor entry should have visitor_name")
			}
			if e.VehiclePlate == nil || *e.VehiclePlate == "" {
				t.Error("visitor entry should have vehicle_plate")
			}
		}
	}
	if !hasAssignment {
		t.Error("expected at least one assignment entry")
	}
	if !hasVisitor {
		t.Error("expected at least one visitor entry")
	}
}
