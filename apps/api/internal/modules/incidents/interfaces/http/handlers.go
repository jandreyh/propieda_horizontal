package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/incidents/application/dto"
	"github.com/saas-ph/api/internal/modules/incidents/application/usecases"
	"github.com/saas-ph/api/internal/modules/incidents/domain"
	"github.com/saas-ph/api/internal/modules/incidents/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger      *slog.Logger
	Incidents   domain.IncidentRepository
	Attachments domain.AttachmentRepository
	History     domain.StatusHistoryRepository
	Assignments domain.IncidentAssignmentRepository
	Outbox      domain.OutboxRepository
	TxRunner    usecases.TxRunner
	Now         func() time.Time
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

// --- Report ---

// reportIncident POST /incidents
func (h *handlers) reportIncident(w http.ResponseWriter, r *http.Request) {
	var body dto.ReportIncidentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.ReportIncident{
		Incidents: h.deps.Incidents,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ReportIncidentInput{
		IncidentType:   entities.IncidentType(body.IncidentType),
		Severity:       entities.Severity(body.Severity),
		Title:          body.Title,
		Description:    body.Description,
		StructureID:    body.StructureID,
		LocationDetail: body.LocationDetail,
		ActorID:        actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, incidentToDTO(out))
}

// --- List ---

// listIncidents GET /incidents?status=...&severity=...&mine=true
func (h *handlers) listIncidents(w http.ResponseWriter, r *http.Request) {
	var filter usecases.ListIncidentsFilter

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := entities.IncidentStatus(statusStr)
		if !status.IsValid() {
			h.fail(w, r, apperrors.BadRequest("status: invalid incident status"))
			return
		}
		filter.Status = &status
	}
	if sevStr := r.URL.Query().Get("severity"); sevStr != "" {
		sev := entities.Severity(sevStr)
		if !sev.IsValid() {
			h.fail(w, r, apperrors.BadRequest("severity: invalid severity"))
			return
		}
		filter.Severity = &sev
	}
	if r.URL.Query().Get("mine") == "true" {
		actorID := actorIDFromCtx(r)
		if actorID != "" {
			filter.ReportedByUserID = &actorID
		}
	}

	uc := usecases.ListIncidents{Incidents: h.deps.Incidents}
	out, err := uc.Execute(r.Context(), filter)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListIncidentsResponse{
		Items: make([]dto.IncidentResponse, 0, len(out)),
		Total: len(out),
	}
	for _, inc := range out {
		resp.Items = append(resp.Items, incidentToDTO(inc))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Get ---

// getIncident GET /incidents/{id}
func (h *handlers) getIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetIncident{Incidents: h.deps.Incidents}
	out, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incidentToDTO(out))
}

// --- Assign ---

// assignIncident POST /incidents/{id}/assign
func (h *handlers) assignIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.AssignIncidentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.AssignIncident{
		Incidents:   h.deps.Incidents,
		Assignments: h.deps.Assignments,
		History:     h.deps.History,
		Outbox:      h.deps.Outbox,
		TxRunner:    h.deps.TxRunner,
		Now:         h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.AssignIncidentInput{
		IncidentID:       id,
		AssignedToUserID: body.AssignedToUserID,
		ActorID:          actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incidentToDTO(out))
}

// --- Start ---

// startIncident POST /incidents/{id}/start
func (h *handlers) startIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.StartIncident{
		Incidents: h.deps.Incidents,
		History:   h.deps.History,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), id, actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incidentToDTO(out))
}

// --- Resolve ---

// resolveIncident POST /incidents/{id}/resolve
func (h *handlers) resolveIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.ResolveIncidentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.ResolveIncident{
		Incidents: h.deps.Incidents,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.ResolveIncidentInput{
		IncidentID:      id,
		ResolutionNotes: body.ResolutionNotes,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incidentToDTO(out))
}

// --- Close ---

// closeIncident POST /incidents/{id}/close
func (h *handlers) closeIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.CloseIncidentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CloseIncident{
		Incidents: h.deps.Incidents,
		History:   h.deps.History,
		Outbox:    h.deps.Outbox,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CloseIncidentInput{
		IncidentID:      id,
		ResolutionNotes: body.ResolutionNotes,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incidentToDTO(out))
}

// --- Cancel ---

// cancelIncident POST /incidents/{id}/cancel
func (h *handlers) cancelIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CancelIncident{
		Incidents: h.deps.Incidents,
		History:   h.deps.History,
		TxRunner:  h.deps.TxRunner,
		Now:       h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), id, actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incidentToDTO(out))
}

// --- Attachments ---

// addAttachment POST /incidents/{id}/attachments
func (h *handlers) addAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.AddAttachmentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.AddAttachment{
		Incidents:   h.deps.Incidents,
		Attachments: h.deps.Attachments,
	}
	out, err := uc.Execute(r.Context(), usecases.AddAttachmentInput{
		IncidentID: id,
		URL:        body.URL,
		MimeType:   body.MimeType,
		SizeBytes:  body.SizeBytes,
		ActorID:    actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, attachmentToDTO(out))
}

// --- Status History ---

// getStatusHistory GET /incidents/{id}/history
func (h *handlers) getStatusHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetStatusHistory{
		Incidents: h.deps.Incidents,
		History:   h.deps.History,
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
			h.deps.Logger.ErrorContext(r.Context(), "incidents: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "incidents: unexpected error",
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

func incidentToDTO(i entities.Incident) dto.IncidentResponse {
	return dto.IncidentResponse{
		ID:               i.ID,
		IncidentType:     string(i.IncidentType),
		Severity:         string(i.Severity),
		Title:            i.Title,
		Description:      i.Description,
		ReportedByUserID: i.ReportedByUserID,
		ReportedAt:       dto.FormatTime(i.ReportedAt),
		StructureID:      i.StructureID,
		LocationDetail:   i.LocationDetail,
		AssignedToUserID: i.AssignedToUserID,
		AssignedAt:       dto.FormatTimePtr(i.AssignedAt),
		StartedAt:        dto.FormatTimePtr(i.StartedAt),
		ResolvedAt:       dto.FormatTimePtr(i.ResolvedAt),
		ClosedAt:         dto.FormatTimePtr(i.ClosedAt),
		CancelledAt:      dto.FormatTimePtr(i.CancelledAt),
		ResolutionNotes:  i.ResolutionNotes,
		Escalated:        i.Escalated,
		SLAAssignDueAt:   dto.FormatTimePtr(i.SLAAssignDueAt),
		SLAResolveDueAt:  dto.FormatTimePtr(i.SLAResolveDueAt),
		Status:           string(i.Status),
		CreatedAt:        dto.FormatTime(i.CreatedAt),
		UpdatedAt:        dto.FormatTime(i.UpdatedAt),
		Version:          i.Version,
	}
}

func attachmentToDTO(a entities.Attachment) dto.AttachmentResponse {
	return dto.AttachmentResponse{
		ID:         a.ID,
		IncidentID: a.IncidentID,
		URL:        a.URL,
		MimeType:   a.MimeType,
		SizeBytes:  a.SizeBytes,
		UploadedBy: a.UploadedBy,
		CreatedAt:  dto.FormatTime(a.CreatedAt),
	}
}

func statusHistoryToDTO(sh entities.StatusHistory) dto.StatusHistoryResponse {
	return dto.StatusHistoryResponse{
		ID:                   sh.ID,
		IncidentID:           sh.IncidentID,
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
