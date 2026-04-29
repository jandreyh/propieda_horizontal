package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/reservations/application/dto"
	"github.com/saas-ph/api/internal/modules/reservations/application/usecases"
	"github.com/saas-ph/api/internal/modules/reservations/domain"
	"github.com/saas-ph/api/internal/modules/reservations/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger       *slog.Logger
	CommonAreas  domain.CommonAreaRepository
	Blackouts    domain.BlackoutRepository
	Reservations domain.ReservationRepository
	History      domain.StatusHistoryRepository
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

// --- Common Areas ---

// createCommonArea POST /common-areas
func (h *handlers) createCommonArea(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateCommonAreaRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}

	slotDuration := int32(60)
	if body.SlotDurationMinutes != nil {
		slotDuration = *body.SlotDurationMinutes
	}
	costPerUse := float64(0)
	if body.CostPerUse != nil {
		costPerUse = *body.CostPerUse
	}
	secDeposit := float64(0)
	if body.SecurityDeposit != nil {
		secDeposit = *body.SecurityDeposit
	}
	requiresApproval := false
	if body.RequiresApproval != nil {
		requiresApproval = *body.RequiresApproval
	}
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	uc := usecases.CreateCommonArea{
		CommonAreas: h.deps.CommonAreas,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateCommonAreaInput{
		Code:                body.Code,
		Name:                body.Name,
		Kind:                entities.CommonAreaKind(body.Kind),
		MaxCapacity:         body.MaxCapacity,
		OpeningTime:         body.OpeningTime,
		ClosingTime:         body.ClosingTime,
		SlotDurationMinutes: slotDuration,
		CostPerUse:          costPerUse,
		SecurityDeposit:     secDeposit,
		RequiresApproval:    requiresApproval,
		IsActive:            isActive,
		Description:         body.Description,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, commonAreaToDTO(out))
}

// updateCommonArea PUT /common-areas/{id}
func (h *handlers) updateCommonArea(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.UpdateCommonAreaRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}

	slotDuration := int32(60)
	if body.SlotDurationMinutes != nil {
		slotDuration = *body.SlotDurationMinutes
	}
	costPerUse := float64(0)
	if body.CostPerUse != nil {
		costPerUse = *body.CostPerUse
	}
	secDeposit := float64(0)
	if body.SecurityDeposit != nil {
		secDeposit = *body.SecurityDeposit
	}
	requiresApproval := false
	if body.RequiresApproval != nil {
		requiresApproval = *body.RequiresApproval
	}
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	actorID := actorIDFromCtx(r)
	uc := usecases.UpdateCommonArea{
		CommonAreas: h.deps.CommonAreas,
	}
	out, err := uc.Execute(r.Context(), usecases.UpdateCommonAreaInput{
		ID:                  id,
		Code:                body.Code,
		Name:                body.Name,
		Kind:                entities.CommonAreaKind(body.Kind),
		MaxCapacity:         body.MaxCapacity,
		OpeningTime:         body.OpeningTime,
		ClosingTime:         body.ClosingTime,
		SlotDurationMinutes: slotDuration,
		CostPerUse:          costPerUse,
		SecurityDeposit:     secDeposit,
		RequiresApproval:    requiresApproval,
		IsActive:            isActive,
		Description:         body.Description,
		Status:              entities.CommonAreaStatus(body.Status),
		ExpectedVersion:     body.Version,
		ActorID:             actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, commonAreaToDTO(out))
}

// listCommonAreas GET /common-areas
func (h *handlers) listCommonAreas(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListCommonAreas{CommonAreas: h.deps.CommonAreas}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListCommonAreasResponse{
		Items: make([]dto.CommonAreaResponse, 0, len(out)),
		Total: len(out),
	}
	for _, a := range out {
		resp.Items = append(resp.Items, commonAreaToDTO(a))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Blackouts ---

// createBlackout POST /common-areas/{id}/blackouts
func (h *handlers) createBlackout(w http.ResponseWriter, r *http.Request) {
	commonAreaID := chi.URLParam(r, "id")
	var body dto.CreateBlackoutRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}

	fromAt, err := time.Parse(time.RFC3339, body.FromAt)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("from_at: invalid RFC3339 format"))
		return
	}
	toAt, err := time.Parse(time.RFC3339, body.ToAt)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("to_at: invalid RFC3339 format"))
		return
	}

	actorID := actorIDFromCtx(r)
	uc := usecases.CreateBlackout{
		CommonAreas: h.deps.CommonAreas,
		Blackouts:   h.deps.Blackouts,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateBlackoutInput{
		CommonAreaID: commonAreaID,
		FromAt:       fromAt,
		ToAt:         toAt,
		Reason:       body.Reason,
		ActorID:      actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, blackoutToDTO(out))
}

// --- Availability ---

// getAvailability GET /common-areas/{id}/availability?date=YYYY-MM-DD
func (h *handlers) getAvailability(w http.ResponseWriter, r *http.Request) {
	commonAreaID := chi.URLParam(r, "id")
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

	uc := usecases.GetAvailability{
		CommonAreas:  h.deps.CommonAreas,
		Reservations: h.deps.Reservations,
		Blackouts:    h.deps.Blackouts,
	}
	slots, err := uc.Execute(r.Context(), commonAreaID, date)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	dtoSlots := make([]dto.AvailabilitySlot, 0, len(slots))
	for _, s := range slots {
		dtoSlots = append(dtoSlots, dto.AvailabilitySlot{
			SlotStart:   dto.FormatTime(s.SlotStart),
			SlotEnd:     dto.FormatTime(s.SlotEnd),
			IsAvailable: s.IsAvailable,
		})
	}
	writeJSON(w, http.StatusOK, dto.AvailabilityResponse{
		CommonAreaID: commonAreaID,
		Date:         dateStr,
		Slots:        dtoSlots,
	})
}

// --- Reservations ---

// createReservation POST /reservations
func (h *handlers) createReservation(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateReservationRequest
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

	// Read Idempotency-Key header if not in body.
	idempotencyKey := body.IdempotencyKey
	if idempotencyKey == nil || *idempotencyKey == "" {
		headerKey := r.Header.Get("Idempotency-Key")
		if headerKey != "" {
			idempotencyKey = &headerKey
		}
	}

	actorID := actorIDFromCtx(r)
	uc := usecases.CreateReservation{
		CommonAreas:  h.deps.CommonAreas,
		Reservations: h.deps.Reservations,
		Blackouts:    h.deps.Blackouts,
		History:      h.deps.History,
		Outbox:       h.deps.Outbox,
		TxRunner:     h.deps.TxRunner,
		Now:          h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateReservationInput{
		CommonAreaID:   body.CommonAreaID,
		UnitID:         body.UnitID,
		RequestedBy:    actorID,
		SlotStartAt:    slotStart,
		SlotEndAt:      slotEnd,
		AttendeesCount: body.AttendeesCount,
		Notes:          body.Notes,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, reservationToDTO(out))
}

// cancelReservation POST /reservations/{id}/cancel
func (h *handlers) cancelReservation(w http.ResponseWriter, r *http.Request) {
	reservationID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CancelReservation{
		Reservations: h.deps.Reservations,
		History:      h.deps.History,
		Outbox:       h.deps.Outbox,
		TxRunner:     h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.CancelReservationInput{
		ReservationID: reservationID,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reservationToDTO(out))
}

// approveReservation POST /reservations/{id}/approve
func (h *handlers) approveReservation(w http.ResponseWriter, r *http.Request) {
	reservationID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.ApproveReservation{
		Reservations: h.deps.Reservations,
		History:      h.deps.History,
		Outbox:       h.deps.Outbox,
		TxRunner:     h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.ApproveReservationInput{
		ReservationID: reservationID,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reservationToDTO(out))
}

// rejectReservation POST /reservations/{id}/reject
func (h *handlers) rejectReservation(w http.ResponseWriter, r *http.Request) {
	reservationID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.RejectReservation{
		Reservations: h.deps.Reservations,
		History:      h.deps.History,
		Outbox:       h.deps.Outbox,
		TxRunner:     h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.RejectReservationInput{
		ReservationID: reservationID,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reservationToDTO(out))
}

// checkinReservation POST /reservations/{id}/checkin
func (h *handlers) checkinReservation(w http.ResponseWriter, r *http.Request) {
	reservationID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CheckinReservation{
		Reservations: h.deps.Reservations,
		History:      h.deps.History,
		Outbox:       h.deps.Outbox,
		TxRunner:     h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.CheckinReservationInput{
		ReservationID: reservationID,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reservationToDTO(out))
}

// listReservations GET /reservations
func (h *handlers) listReservations(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListReservations{Reservations: h.deps.Reservations}
	out, err := uc.Execute(r.Context())
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

// listMyReservations GET /reservations/mine?unit_id=...
func (h *handlers) listMyReservations(w http.ResponseWriter, r *http.Request) {
	unitID := r.URL.Query().Get("unit_id")
	if unitID == "" {
		h.fail(w, r, apperrors.BadRequest("unit_id query parameter is required"))
		return
	}
	uc := usecases.ListMyReservations{Reservations: h.deps.Reservations}
	out, err := uc.Execute(r.Context(), unitID)
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

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "reservations: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "reservations: unexpected error",
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

func commonAreaToDTO(a entities.CommonArea) dto.CommonAreaResponse {
	return dto.CommonAreaResponse{
		ID:                  a.ID,
		Code:                a.Code,
		Name:                a.Name,
		Kind:                string(a.Kind),
		MaxCapacity:         a.MaxCapacity,
		OpeningTime:         a.OpeningTime,
		ClosingTime:         a.ClosingTime,
		SlotDurationMinutes: a.SlotDurationMinutes,
		CostPerUse:          a.CostPerUse,
		SecurityDeposit:     a.SecurityDeposit,
		RequiresApproval:    a.RequiresApproval,
		IsActive:            a.IsActive,
		Description:         a.Description,
		Status:              string(a.Status),
		CreatedAt:           dto.FormatTime(a.CreatedAt),
		UpdatedAt:           dto.FormatTime(a.UpdatedAt),
		Version:             a.Version,
	}
}

func blackoutToDTO(b entities.ReservationBlackout) dto.BlackoutResponse {
	return dto.BlackoutResponse{
		ID:           b.ID,
		CommonAreaID: b.CommonAreaID,
		FromAt:       dto.FormatTime(b.FromAt),
		ToAt:         dto.FormatTime(b.ToAt),
		Reason:       b.Reason,
		Status:       string(b.Status),
		CreatedAt:    dto.FormatTime(b.CreatedAt),
		Version:      b.Version,
	}
}

func reservationToDTO(rv entities.Reservation) dto.ReservationResponse {
	return dto.ReservationResponse{
		ID:                rv.ID,
		CommonAreaID:      rv.CommonAreaID,
		UnitID:            rv.UnitID,
		RequestedByUserID: rv.RequestedByUserID,
		SlotStartAt:       dto.FormatTime(rv.SlotStartAt),
		SlotEndAt:         dto.FormatTime(rv.SlotEndAt),
		AttendeesCount:    rv.AttendeesCount,
		Cost:              rv.Cost,
		SecurityDeposit:   rv.SecurityDeposit,
		DepositRefunded:   rv.DepositRefunded,
		QRCodeHash:        rv.QRCodeHash,
		Notes:             rv.Notes,
		ApprovedBy:        rv.ApprovedBy,
		ApprovedAt:        dto.FormatTimePtr(rv.ApprovedAt),
		CancelledBy:       rv.CancelledBy,
		CancelledAt:       dto.FormatTimePtr(rv.CancelledAt),
		ConsumedAt:        dto.FormatTimePtr(rv.ConsumedAt),
		Status:            string(rv.Status),
		CreatedAt:         dto.FormatTime(rv.CreatedAt),
		UpdatedAt:         dto.FormatTime(rv.UpdatedAt),
		Version:           rv.Version,
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
