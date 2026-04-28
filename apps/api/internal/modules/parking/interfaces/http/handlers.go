package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/parking/application/dto"
	"github.com/saas-ph/api/internal/modules/parking/application/usecases"
	"github.com/saas-ph/api/internal/modules/parking/domain"
	"github.com/saas-ph/api/internal/modules/parking/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger       *slog.Logger
	Spaces       domain.SpaceRepository
	Assignments  domain.AssignmentRepository
	History      domain.AssignmentHistoryRepository
	Reservations domain.VisitorReservationRepository
	Lotteries    domain.LotteryRunRepository
	Results      domain.LotteryResultRepository
	Outbox       domain.OutboxRepository
	TxRunner     usecases.TxRunner
	Now          func() time.Time
}

// validate completa los defaults razonables (logger, clock).
func (d *Dependencies) validate() {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
}

// handlers agrupa los handlers HTTP construidos a partir de Dependencies.
type handlers struct {
	deps Dependencies
}

func newHandlers(d Dependencies) *handlers {
	d.validate()
	return &handlers{deps: d}
}

// --- Parking Spaces ---

// createSpace POST /parking-spaces
func (h *handlers) createSpace(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateSpaceRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.CreateSpace{
		Spaces: h.deps.Spaces,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateSpaceInput{
		Code:        body.Code,
		Type:        entities.SpaceType(body.Type),
		StructureID: body.StructureID,
		Level:       body.Level,
		Zone:        body.Zone,
		MonthlyFee:  body.MonthlyFee,
		IsVisitor:   body.IsVisitor,
		Notes:       body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, spaceToDTO(out))
}

// updateSpace PUT /parking-spaces/{id}
func (h *handlers) updateSpace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.UpdateSpaceRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.UpdateSpace{
		Spaces: h.deps.Spaces,
	}
	out, err := uc.Execute(r.Context(), usecases.UpdateSpaceInput{
		ID:              id,
		Code:            body.Code,
		Type:            entities.SpaceType(body.Type),
		StructureID:     body.StructureID,
		Level:           body.Level,
		Zone:            body.Zone,
		MonthlyFee:      body.MonthlyFee,
		IsVisitor:       body.IsVisitor,
		Notes:           body.Notes,
		Status:          entities.SpaceStatus(body.Status),
		ExpectedVersion: body.Version,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, spaceToDTO(out))
}

// listSpaces GET /parking-spaces
func (h *handlers) listSpaces(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListSpaces{Spaces: h.deps.Spaces}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListSpacesResponse{
		Items: make([]dto.SpaceResponse, 0, len(out)),
		Total: len(out),
	}
	for _, s := range out {
		resp.Items = append(resp.Items, spaceToDTO(s))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Parking Assignments ---

// assignSpace POST /parking-spaces/{id}/assign
func (h *handlers) assignSpace(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "id")
	var body dto.AssignSpaceRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.AssignSpace{
		Spaces:      h.deps.Spaces,
		Assignments: h.deps.Assignments,
		History:     h.deps.History,
		Outbox:      h.deps.Outbox,
		TxRunner:    h.deps.TxRunner,
		Now:         h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.AssignSpaceInput{
		SpaceID:   spaceID,
		UnitID:    body.UnitID,
		VehicleID: body.VehicleID,
		ActorID:   actorID,
		Notes:     body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, assignmentToDTO(out))
}

// releaseAssignment POST /parking-assignments/{id}/release
func (h *handlers) releaseAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	var body dto.ReleaseAssignmentRequest
	if err := decodeJSON(r, &body); err != nil {
		// Allow empty body for release.
		if !errors.Is(err, apperrors.Problem{}) {
			body = dto.ReleaseAssignmentRequest{}
		} else {
			h.fail(w, r, err)
			return
		}
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.ReleaseAssignment{
		Assignments: h.deps.Assignments,
		History:     h.deps.History,
		Outbox:      h.deps.Outbox,
		TxRunner:    h.deps.TxRunner,
		Now:         h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ReleaseAssignmentInput{
		AssignmentID: assignmentID,
		ActorID:      actorID,
		Reason:       body.Reason,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, assignmentToDTO(out))
}

// getUnitParking GET /units/{id}/parking
func (h *handlers) getUnitParking(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "id")
	uc := usecases.GetUnitParking{
		Assignments:  h.deps.Assignments,
		Reservations: h.deps.Reservations,
	}
	out, err := uc.Execute(r.Context(), unitID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	assignments := make([]dto.AssignmentResponse, 0, len(out.Assignments))
	for _, a := range out.Assignments {
		assignments = append(assignments, assignmentToDTO(a))
	}
	reservations := make([]dto.ReservationResponse, 0, len(out.Reservations))
	for _, rv := range out.Reservations {
		reservations = append(reservations, reservationToDTO(rv))
	}
	writeJSON(w, http.StatusOK, dto.UnitParkingResponse{
		Assignments:  assignments,
		Reservations: reservations,
	})
}

// --- Visitor Reservations ---

// createVisitorReservation POST /parking-visitor-reservations
func (h *handlers) createVisitorReservation(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateVisitorReservationRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	slotStart, err := time.Parse(time.RFC3339, body.SlotStartAt)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("slot_start_at: invalid RFC3339 format"))
		return
	}
	slotEnd, err := time.Parse(time.RFC3339, body.SlotEndAt)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("slot_end_at: invalid RFC3339 format"))
		return
	}

	actorID := actorIDFromCtx(r)
	uc := usecases.CreateVisitorReservation{
		Spaces:       h.deps.Spaces,
		Reservations: h.deps.Reservations,
		Outbox:       h.deps.Outbox,
		TxRunner:     h.deps.TxRunner,
		Now:          h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateVisitorReservationInput{
		ParkingSpaceID:  body.ParkingSpaceID,
		UnitID:          body.UnitID,
		RequestedBy:     actorID,
		VisitorName:     body.VisitorName,
		VisitorDocument: body.VisitorDocument,
		VehiclePlate:    body.VehiclePlate,
		SlotStartAt:     slotStart,
		SlotEndAt:       slotEnd,
		IdempotencyKey:  body.IdempotencyKey,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, reservationToDTO(out))
}

// cancelVisitorReservation POST /parking-visitor-reservations/{id}/cancel
func (h *handlers) cancelVisitorReservation(w http.ResponseWriter, r *http.Request) {
	reservationID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CancelVisitorReservation{
		Reservations: h.deps.Reservations,
	}
	out, err := uc.Execute(r.Context(), usecases.CancelVisitorReservationInput{
		ReservationID: reservationID,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reservationToDTO(out))
}

// listVisitorReservations GET /parking-visitor-reservations?date=...
func (h *handlers) listVisitorReservations(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		h.fail(w, r, apperrors.BadRequest("date query parameter is required (YYYY-MM-DD)"))
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("date: invalid format, expected YYYY-MM-DD"))
		return
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	uc := usecases.ListVisitorReservations{
		Reservations: h.deps.Reservations,
	}
	out, err := uc.Execute(r.Context(), start, end)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListReservationsResponse{
		Items: make([]dto.ReservationResponse, 0, len(out)),
		Total: len(out),
	}
	for _, rv := range out {
		resp.Items = append(resp.Items, reservationToDTO(rv))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Lotteries ---

// runLottery POST /parking-lotteries/run
func (h *handlers) runLottery(w http.ResponseWriter, r *http.Request) {
	var body dto.RunLotteryRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.RunLottery{
		Spaces:    h.deps.Spaces,
		Lotteries: h.deps.Lotteries,
		Results:   h.deps.Results,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.RunLotteryInput{
		Name:          body.Name,
		Seed:          body.Seed,
		SpaceIDs:      body.SpaceIDs,
		EligibleUnits: body.EligibleUnits,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, lotteryResultsToDTO(out))
}

// getLotteryResults GET /parking-lotteries/{id}/results
func (h *handlers) getLotteryResults(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "id")
	uc := usecases.GetLotteryResults{
		Lotteries: h.deps.Lotteries,
		Results:   h.deps.Results,
	}
	out, err := uc.Execute(r.Context(), runID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, lotteryResultsToDTO(out))
}

// --- Guard ---

// guardParkingToday GET /guard/parking/today
func (h *handlers) guardParkingToday(w http.ResponseWriter, r *http.Request) {
	uc := usecases.GuardParkingToday{
		Spaces:       h.deps.Spaces,
		Assignments:  h.deps.Assignments,
		Reservations: h.deps.Reservations,
		Now:          h.deps.Now,
	}
	entries, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	now := h.deps.Now()
	dateStr := now.Format("2006-01-02")
	dtoEntries := make([]dto.GuardParkingEntryResponse, 0, len(entries))
	for _, e := range entries {
		dtoEntries = append(dtoEntries, guardEntryToDTO(e))
	}
	writeJSON(w, http.StatusOK, dto.GuardParkingTodayResponse{
		Date:    dateStr,
		Entries: dtoEntries,
		Total:   len(dtoEntries),
	})
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "parking: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "parking: unexpected error",
		slog.String("path", r.URL.Path),
		slog.String("err", err.Error()))
	apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperrors.BadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}

// --- Entity-to-DTO mapping functions ---

func spaceToDTO(s entities.ParkingSpace) dto.SpaceResponse {
	return dto.SpaceResponse{
		ID:          s.ID,
		Code:        s.Code,
		Type:        string(s.Type),
		StructureID: s.StructureID,
		Level:       s.Level,
		Zone:        s.Zone,
		MonthlyFee:  s.MonthlyFee,
		IsVisitor:   s.IsVisitor,
		Notes:       s.Notes,
		Status:      string(s.Status),
		CreatedAt:   dto.FormatTime(s.CreatedAt),
		UpdatedAt:   dto.FormatTime(s.UpdatedAt),
		Version:     s.Version,
	}
}

func assignmentToDTO(a entities.ParkingAssignment) dto.AssignmentResponse {
	return dto.AssignmentResponse{
		ID:               a.ID,
		ParkingSpaceID:   a.ParkingSpaceID,
		UnitID:           a.UnitID,
		VehicleID:        a.VehicleID,
		AssignedByUserID: a.AssignedByUserID,
		SinceDate:        dto.FormatTime(a.SinceDate),
		UntilDate:        dto.FormatTimePtr(a.UntilDate),
		Notes:            a.Notes,
		Status:           string(a.Status),
		CreatedAt:        dto.FormatTime(a.CreatedAt),
		UpdatedAt:        dto.FormatTime(a.UpdatedAt),
		Version:          a.Version,
	}
}

func reservationToDTO(rv entities.VisitorReservation) dto.ReservationResponse {
	return dto.ReservationResponse{
		ID:              rv.ID,
		ParkingSpaceID:  rv.ParkingSpaceID,
		UnitID:          rv.UnitID,
		RequestedBy:     rv.RequestedBy,
		VisitorName:     rv.VisitorName,
		VisitorDocument: rv.VisitorDocument,
		VehiclePlate:    rv.VehiclePlate,
		SlotStartAt:     dto.FormatTime(rv.SlotStartAt),
		SlotEndAt:       dto.FormatTime(rv.SlotEndAt),
		Status:          string(rv.Status),
		CreatedAt:       dto.FormatTime(rv.CreatedAt),
		UpdatedAt:       dto.FormatTime(rv.UpdatedAt),
		Version:         rv.Version,
	}
}

func lotteryRunToDTO(run entities.LotteryRun) dto.LotteryRunResponse {
	return dto.LotteryRunResponse{
		ID:         run.ID,
		Name:       run.Name,
		SeedHash:   run.SeedHash,
		ExecutedAt: dto.FormatTime(run.ExecutedAt),
		ExecutedBy: run.ExecutedBy,
		Status:     string(run.Status),
		CreatedAt:  dto.FormatTime(run.CreatedAt),
		UpdatedAt:  dto.FormatTime(run.UpdatedAt),
		Version:    run.Version,
	}
}

func lotteryResultToDTO(result entities.LotteryResult) dto.LotteryResultResponse {
	return dto.LotteryResultResponse{
		ID:             result.ID,
		LotteryRunID:   result.LotteryRunID,
		UnitID:         result.UnitID,
		ParkingSpaceID: result.ParkingSpaceID,
		Position:       result.Position,
		Status:         string(result.Status),
	}
}

func lotteryResultsToDTO(out usecases.RunLotteryResult) dto.LotteryResultsResponse {
	results := make([]dto.LotteryResultResponse, 0, len(out.Results))
	for _, r := range out.Results {
		results = append(results, lotteryResultToDTO(r))
	}
	return dto.LotteryResultsResponse{
		Run:     lotteryRunToDTO(out.Run),
		Results: results,
	}
}

func guardEntryToDTO(e usecases.GuardParkingEntry) dto.GuardParkingEntryResponse {
	var slotStart, slotEnd *string
	if e.SlotStartAt != nil {
		s := dto.FormatTime(*e.SlotStartAt)
		slotStart = &s
	}
	if e.SlotEndAt != nil {
		s := dto.FormatTime(*e.SlotEndAt)
		slotEnd = &s
	}
	return dto.GuardParkingEntryResponse{
		SpaceCode:    e.SpaceCode,
		SpaceType:    string(e.SpaceType),
		UnitID:       e.UnitID,
		VehiclePlate: e.VehiclePlate,
		VisitorName:  e.VisitorName,
		SlotStartAt:  slotStart,
		SlotEndAt:    slotEnd,
		EntryType:    e.EntryType,
	}
}

// actorCtxKey clave de contexto para el actor (user_id) que origina la
// peticion.
type actorCtxKey struct{}

// WithActorID es helper para inyectar el actor desde un middleware
// externo (test o capa auth).
func WithActorID(r *http.Request, actorID string) *http.Request {
	if actorID == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), actorCtxKey{}, actorID))
}

func actorIDFromCtx(r *http.Request) string {
	if v, ok := r.Context().Value(actorCtxKey{}).(string); ok {
		return v
	}
	return ""
}
