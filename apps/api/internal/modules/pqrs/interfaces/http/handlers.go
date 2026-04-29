package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/pqrs/application/dto"
	"github.com/saas-ph/api/internal/modules/pqrs/application/usecases"
	"github.com/saas-ph/api/internal/modules/pqrs/domain"
	"github.com/saas-ph/api/internal/modules/pqrs/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger     *slog.Logger
	Categories domain.CategoryRepository
	Tickets    domain.TicketRepository
	Responses  domain.ResponseRepository
	History    domain.StatusHistoryRepository
	Outbox     domain.OutboxRepository
	TxRunner   usecases.TxRunner
	Now        func() time.Time
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

// --- Categories ---

// createCategory POST /pqrs/categories
func (h *handlers) createCategory(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateCategoryRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateCategory{
		Categories: h.deps.Categories,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateCategoryInput{
		Code:                  body.Code,
		Name:                  body.Name,
		DefaultAssigneeRoleID: body.DefaultAssigneeRoleID,
		ActorID:               actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, categoryToDTO(out))
}

// listCategories GET /pqrs/categories
func (h *handlers) listCategories(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListCategories{Categories: h.deps.Categories}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListCategoriesResponse{
		Items: make([]dto.CategoryResponse, 0, len(out)),
		Total: len(out),
	}
	for _, c := range out {
		resp.Items = append(resp.Items, categoryToDTO(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// updateCategory PATCH /pqrs/categories/{id}
func (h *handlers) updateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.UpdateCategoryRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.UpdateCategory{
		Categories: h.deps.Categories,
	}
	out, err := uc.Execute(r.Context(), usecases.UpdateCategoryInput{
		ID:                    id,
		Code:                  body.Code,
		Name:                  body.Name,
		DefaultAssigneeRoleID: body.DefaultAssigneeRoleID,
		Status:                entities.CategoryStatus(body.Status),
		ExpectedVersion:       body.Version,
		ActorID:               actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, categoryToDTO(out))
}

// --- Tickets ---

// fileTicket POST /pqrs
func (h *handlers) fileTicket(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateTicketRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.FileTicket{
		Tickets:  h.deps.Tickets,
		History:  h.deps.History,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
		Now:      h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.FileTicketInput{
		PQRType:     entities.PQRType(body.PQRType),
		CategoryID:  body.CategoryID,
		Subject:     body.Subject,
		Body:        body.Body,
		IsAnonymous: body.IsAnonymous,
		ActorID:     actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, ticketToDTO(out))
}

// listTickets GET /pqrs
func (h *handlers) listTickets(w http.ResponseWriter, r *http.Request) {
	var input usecases.ListTicketsInput

	if s := r.URL.Query().Get("status"); s != "" {
		st := entities.TicketStatus(s)
		input.Status = &st
	}
	if t := r.URL.Query().Get("type"); t != "" {
		pt := entities.PQRType(t)
		input.PQRType = &pt
	}
	if mine := r.URL.Query().Get("mine"); mine == "true" {
		actorID := actorIDFromCtx(r)
		if actorID != "" {
			input.RequesterUserID = &actorID
		}
	}

	uc := usecases.ListTickets{Tickets: h.deps.Tickets}
	out, err := uc.Execute(r.Context(), input)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListTicketsResponse{
		Items: make([]dto.TicketResponse, 0, len(out)),
		Total: len(out),
	}
	for _, t := range out {
		resp.Items = append(resp.Items, ticketToDTO(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

// getTicket GET /pqrs/{id}
func (h *handlers) getTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetTicket{Tickets: h.deps.Tickets}
	out, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// assignTicket POST /pqrs/{id}/assign
func (h *handlers) assignTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.AssignTicketRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.AssignTicket{
		Tickets:  h.deps.Tickets,
		History:  h.deps.History,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.AssignTicketInput{
		TicketID:         id,
		AssignedToUserID: body.AssignedToUserID,
		ActorID:          actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// startStudy POST /pqrs/{id}/start-study
func (h *handlers) startStudy(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.StartStudy{
		Tickets:  h.deps.Tickets,
		History:  h.deps.History,
		TxRunner: h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.StartStudyInput{
		TicketID: id,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// respondTicket POST /pqrs/{id}/respond
func (h *handlers) respondTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.RespondTicketRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.RespondTicket{
		Tickets:   h.deps.Tickets,
		Responses: h.deps.Responses,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.RespondTicketInput{
		TicketID: id,
		Body:     body.Body,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// addNote POST /pqrs/{id}/notes
func (h *handlers) addNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.AddNoteRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.AddNote{
		Tickets:   h.deps.Tickets,
		Responses: h.deps.Responses,
	}
	out, err := uc.Execute(r.Context(), usecases.AddNoteInput{
		TicketID: id,
		Body:     body.Body,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, responseToDTO(out))
}

// closeTicket POST /pqrs/{id}/close
func (h *handlers) closeTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.CloseTicketRequest
	if err := decodeJSON(r, &body); err != nil {
		// Allow empty body for close.
		if !errors.Is(err, apperrors.Problem{}) {
			body = dto.CloseTicketRequest{}
		} else {
			h.fail(w, r, err)
			return
		}
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CloseTicket{
		Tickets:   h.deps.Tickets,
		Responses: h.deps.Responses,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.CloseTicketInput{
		TicketID: id,
		Rating:   body.Rating,
		Feedback: body.Feedback,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// escalateTicket POST /pqrs/{id}/escalate
func (h *handlers) escalateTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.EscalateTicketRequest
	if err := decodeJSON(r, &body); err != nil {
		if !errors.Is(err, apperrors.Problem{}) {
			body = dto.EscalateTicketRequest{}
		} else {
			h.fail(w, r, err)
			return
		}
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.EscalateTicket{
		Tickets:  h.deps.Tickets,
		History:  h.deps.History,
		TxRunner: h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.EscalateTicketInput{
		TicketID: id,
		Notes:    body.Notes,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// cancelTicket POST /pqrs/{id}/cancel
func (h *handlers) cancelTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.CancelTicketRequest
	if err := decodeJSON(r, &body); err != nil {
		if !errors.Is(err, apperrors.Problem{}) {
			body = dto.CancelTicketRequest{}
		} else {
			h.fail(w, r, err)
			return
		}
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CancelTicket{
		Tickets:  h.deps.Tickets,
		History:  h.deps.History,
		TxRunner: h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.CancelTicketInput{
		TicketID: id,
		Notes:    body.Notes,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ticketToDTO(out))
}

// getTicketHistory GET /pqrs/{id}/history
func (h *handlers) getTicketHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetTicketHistory{
		Tickets: h.deps.Tickets,
		History: h.deps.History,
	}
	out, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListStatusHistoryResponse{
		Items: make([]dto.StatusHistoryResponse, 0, len(out)),
		Total: len(out),
	}
	for _, sh := range out {
		resp.Items = append(resp.Items, statusHistoryToDTO(sh))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "pqrs: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "pqrs: unexpected error",
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

func categoryToDTO(c entities.Category) dto.CategoryResponse {
	return dto.CategoryResponse{
		ID:                    c.ID,
		Code:                  c.Code,
		Name:                  c.Name,
		DefaultAssigneeRoleID: c.DefaultAssigneeRoleID,
		Status:                string(c.Status),
		CreatedAt:             dto.FormatTime(c.CreatedAt),
		UpdatedAt:             dto.FormatTime(c.UpdatedAt),
		Version:               c.Version,
	}
}

func ticketToDTO(t entities.Ticket) dto.TicketResponse {
	return dto.TicketResponse{
		ID:                t.ID,
		TicketYear:        t.TicketYear,
		SerialNumber:      t.SerialNumber,
		PQRType:           string(t.PQRType),
		CategoryID:        t.CategoryID,
		Subject:           t.Subject,
		Body:              t.Body,
		RequesterUserID:   t.RequesterUserID,
		AssignedToUserID:  t.AssignedToUserID,
		AssignedAt:        dto.FormatTimePtr(t.AssignedAt),
		RespondedAt:       dto.FormatTimePtr(t.RespondedAt),
		ClosedAt:          dto.FormatTimePtr(t.ClosedAt),
		EscalatedAt:       dto.FormatTimePtr(t.EscalatedAt),
		CancelledAt:       dto.FormatTimePtr(t.CancelledAt),
		SLADueAt:          dto.FormatTimePtr(t.SLADueAt),
		RequesterRating:   t.RequesterRating,
		RequesterFeedback: t.RequesterFeedback,
		IsAnonymous:       t.IsAnonymous,
		Status:            string(t.Status),
		CreatedAt:         dto.FormatTime(t.CreatedAt),
		UpdatedAt:         dto.FormatTime(t.UpdatedAt),
		Version:           t.Version,
	}
}

func responseToDTO(r entities.Response) dto.ResponseResponse {
	return dto.ResponseResponse{
		ID:                r.ID,
		TicketID:          r.TicketID,
		ResponseType:      string(r.ResponseType),
		Body:              r.Body,
		RespondedByUserID: r.RespondedByUserID,
		RespondedAt:       dto.FormatTime(r.RespondedAt),
		CreatedAt:         dto.FormatTime(r.CreatedAt),
	}
}

func statusHistoryToDTO(sh entities.StatusHistory) dto.StatusHistoryResponse {
	return dto.StatusHistoryResponse{
		ID:                   sh.ID,
		TicketID:             sh.TicketID,
		FromStatus:           sh.FromStatus,
		ToStatus:             sh.ToStatus,
		TransitionedByUserID: sh.TransitionedByUserID,
		TransitionedAt:       dto.FormatTime(sh.TransitionedAt),
		Notes:                sh.Notes,
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
