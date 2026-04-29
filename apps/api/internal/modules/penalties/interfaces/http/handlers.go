package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/penalties/application/dto"
	"github.com/saas-ph/api/internal/modules/penalties/application/usecases"
	"github.com/saas-ph/api/internal/modules/penalties/domain"
	"github.com/saas-ph/api/internal/modules/penalties/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger    *slog.Logger
	Catalog   domain.CatalogRepository
	Penalties domain.PenaltyRepository
	Appeals   domain.AppealRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  usecases.TxRunner
	Now       func() time.Time
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

// --- Penalty Catalog ---

// createCatalogEntry POST /penalty-catalog
func (h *handlers) createCatalogEntry(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateCatalogRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateCatalogEntry{
		Catalog: h.deps.Catalog,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateCatalogEntryInput{
		Code:                     body.Code,
		Name:                     body.Name,
		Description:              body.Description,
		DefaultSanctionType:      entities.SanctionType(body.DefaultSanctionType),
		BaseAmount:               body.BaseAmount,
		RecurrenceMultiplier:     body.RecurrenceMultiplier,
		RecurrenceCAPMultiplier:  body.RecurrenceCAPMultiplier,
		RequiresCouncilThreshold: body.RequiresCouncilThreshold,
		ActorID:                  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, catalogToDTO(out))
}

// updateCatalogEntry PATCH /penalty-catalog/{id}
func (h *handlers) updateCatalogEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.UpdateCatalogRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.UpdateCatalogEntry{
		Catalog: h.deps.Catalog,
	}
	out, err := uc.Execute(r.Context(), usecases.UpdateCatalogEntryInput{
		ID:                       id,
		Code:                     body.Code,
		Name:                     body.Name,
		Description:              body.Description,
		DefaultSanctionType:      entities.SanctionType(body.DefaultSanctionType),
		BaseAmount:               body.BaseAmount,
		RecurrenceMultiplier:     body.RecurrenceMultiplier,
		RecurrenceCAPMultiplier:  body.RecurrenceCAPMultiplier,
		RequiresCouncilThreshold: body.RequiresCouncilThreshold,
		Status:                   entities.CatalogStatus(body.Status),
		ExpectedVersion:          body.Version,
		ActorID:                  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, catalogToDTO(out))
}

// listCatalog GET /penalty-catalog
func (h *handlers) listCatalog(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListCatalog{Catalog: h.deps.Catalog}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListCatalogResponse{
		Items: make([]dto.CatalogResponse, 0, len(out)),
		Total: len(out),
	}
	for _, c := range out {
		resp.Items = append(resp.Items, catalogToDTO(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Penalties ---

// imposePenalty POST /penalties
func (h *handlers) imposePenalty(w http.ResponseWriter, r *http.Request) {
	var body dto.ImposePenaltyRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)

	var sanctionType *entities.SanctionType
	if body.SanctionType != nil {
		st := entities.SanctionType(*body.SanctionType)
		sanctionType = &st
	}

	uc := usecases.ImposePenalty{
		Catalog:   h.deps.Catalog,
		Penalties: h.deps.Penalties,
		History:   h.deps.History,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ImposePenaltyInput{
		CatalogID:        body.CatalogID,
		DebtorUserID:     body.DebtorUserID,
		UnitID:           body.UnitID,
		SourceIncidentID: body.SourceIncidentID,
		SanctionType:     sanctionType,
		Reason:           body.Reason,
		IdempotencyKey:   body.IdempotencyKey,
		ActorID:          actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, penaltyToDTO(out))
}

// notifyPenalty POST /penalties/{id}/notify
func (h *handlers) notifyPenalty(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.NotifyPenalty{
		Penalties: h.deps.Penalties,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.NotifyPenaltyInput{
		PenaltyID: penaltyID,
		ActorID:   actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, penaltyToDTO(out))
}

// councilApprovePenalty POST /penalties/{id}/council-approve
func (h *handlers) councilApprovePenalty(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CouncilApprovePenalty{
		Penalties: h.deps.Penalties,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CouncilApprovePenaltyInput{
		PenaltyID: penaltyID,
		ActorID:   actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, penaltyToDTO(out))
}

// confirmPenalty POST /penalties/{id}/confirm
func (h *handlers) confirmPenalty(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.ConfirmPenalty{
		Penalties: h.deps.Penalties,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ConfirmPenaltyInput{
		PenaltyID: penaltyID,
		ActorID:   actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, penaltyToDTO(out))
}

// settlePenalty POST /penalties/{id}/settle
func (h *handlers) settlePenalty(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.SettlePenalty{
		Penalties: h.deps.Penalties,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.SettlePenaltyInput{
		PenaltyID: penaltyID,
		ActorID:   actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, penaltyToDTO(out))
}

// cancelPenalty POST /penalties/{id}/cancel
func (h *handlers) cancelPenalty(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CancelPenalty{
		Penalties: h.deps.Penalties,
		History:   h.deps.History,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CancelPenaltyInput{
		PenaltyID: penaltyID,
		ActorID:   actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, penaltyToDTO(out))
}

// submitAppeal POST /penalties/{id}/appeals
func (h *handlers) submitAppeal(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	var body dto.SubmitAppealRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.SubmitAppeal{
		Penalties: h.deps.Penalties,
		Appeals:   h.deps.Appeals,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.SubmitAppealInput{
		PenaltyID: penaltyID,
		Grounds:   body.Grounds,
		ActorID:   actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, appealToDTO(out))
}

// resolveAppeal POST /penalties/{id}/appeals/{aid}/resolve
func (h *handlers) resolveAppeal(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	appealID := chi.URLParam(r, "aid")
	var body dto.ResolveAppealRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.ResolveAppeal{
		Penalties: h.deps.Penalties,
		Appeals:   h.deps.Appeals,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ResolveAppealInput{
		PenaltyID:       penaltyID,
		AppealID:        appealID,
		Resolution:      body.Resolution,
		NewAppealStatus: entities.AppealStatus(body.Status),
		ExpectedVersion: body.Version,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, appealToDTO(out))
}

// listPenalties GET /penalties
func (h *handlers) listPenalties(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListPenalties{Penalties: h.deps.Penalties}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListPenaltiesResponse{
		Items: make([]dto.PenaltyResponse, 0, len(out)),
		Total: len(out),
	}
	for _, p := range out {
		resp.Items = append(resp.Items, penaltyToDTO(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// getPenaltyHistory GET /penalties/{id}/history
func (h *handlers) getPenaltyHistory(w http.ResponseWriter, r *http.Request) {
	penaltyID := chi.URLParam(r, "id")
	uc := usecases.GetPenaltyHistory{
		Penalties: h.deps.Penalties,
		History:   h.deps.History,
	}
	out, err := uc.Execute(r.Context(), penaltyID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListStatusHistoryResponse{
		Items: make([]dto.StatusHistoryResponse, 0, len(out)),
		Total: len(out),
	}
	for _, entry := range out {
		resp.Items = append(resp.Items, statusHistoryToDTO(entry))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "penalties: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "penalties: unexpected error",
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

func catalogToDTO(c entities.PenaltyCatalog) dto.CatalogResponse {
	return dto.CatalogResponse{
		ID:                       c.ID,
		Code:                     c.Code,
		Name:                     c.Name,
		Description:              c.Description,
		DefaultSanctionType:      string(c.DefaultSanctionType),
		BaseAmount:               c.BaseAmount,
		RecurrenceMultiplier:     c.RecurrenceMultiplier,
		RecurrenceCAPMultiplier:  c.RecurrenceCAPMultiplier,
		RequiresCouncilThreshold: c.RequiresCouncilThreshold,
		Status:                   string(c.Status),
		CreatedAt:                dto.FormatTime(c.CreatedAt),
		UpdatedAt:                dto.FormatTime(c.UpdatedAt),
		Version:                  c.Version,
	}
}

func penaltyToDTO(p entities.Penalty) dto.PenaltyResponse {
	return dto.PenaltyResponse{
		ID:                      p.ID,
		CatalogID:               p.CatalogID,
		DebtorUserID:            p.DebtorUserID,
		UnitID:                  p.UnitID,
		SourceIncidentID:        p.SourceIncidentID,
		SanctionType:            string(p.SanctionType),
		Amount:                  p.Amount,
		Reason:                  p.Reason,
		ImposedByUserID:         p.ImposedByUserID,
		NotifiedAt:              dto.FormatTimePtr(p.NotifiedAt),
		AppealDeadlineAt:        dto.FormatTimePtr(p.AppealDeadlineAt),
		ConfirmedAt:             dto.FormatTimePtr(p.ConfirmedAt),
		SettledAt:               dto.FormatTimePtr(p.SettledAt),
		DismissedAt:             dto.FormatTimePtr(p.DismissedAt),
		CancelledAt:             dto.FormatTimePtr(p.CancelledAt),
		RequiresCouncilApproval: p.RequiresCouncilApproval,
		CouncilApprovedByUserID: p.CouncilApprovedByUserID,
		CouncilApprovedAt:       dto.FormatTimePtr(p.CouncilApprovedAt),
		Status:                  string(p.Status),
		CreatedAt:               dto.FormatTime(p.CreatedAt),
		UpdatedAt:               dto.FormatTime(p.UpdatedAt),
		Version:                 p.Version,
	}
}

func appealToDTO(a entities.PenaltyAppeal) dto.AppealResponse {
	return dto.AppealResponse{
		ID:                a.ID,
		PenaltyID:         a.PenaltyID,
		SubmittedByUserID: a.SubmittedByUserID,
		SubmittedAt:       dto.FormatTime(a.SubmittedAt),
		Grounds:           a.Grounds,
		ResolvedByUserID:  a.ResolvedByUserID,
		ResolvedAt:        dto.FormatTimePtr(a.ResolvedAt),
		Resolution:        a.Resolution,
		Status:            string(a.Status),
		CreatedAt:         dto.FormatTime(a.CreatedAt),
		UpdatedAt:         dto.FormatTime(a.UpdatedAt),
		Version:           a.Version,
	}
}

func statusHistoryToDTO(h entities.PenaltyStatusHistory) dto.StatusHistoryResponse {
	return dto.StatusHistoryResponse{
		ID:                   h.ID,
		PenaltyID:            h.PenaltyID,
		FromStatus:           h.FromStatus,
		ToStatus:             h.ToStatus,
		TransitionedByUserID: h.TransitionedByUserID,
		TransitionedAt:       dto.FormatTime(h.TransitionedAt),
		Notes:                h.Notes,
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
