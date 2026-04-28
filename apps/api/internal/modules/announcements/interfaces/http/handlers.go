// Package http contiene los adaptadores HTTP del modulo announcements.
//
// Los handlers traducen request/response al usecase correspondiente y
// emiten errores RFC 7807 via apperrors. NO contienen logica de negocio.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/announcements/application/dto"
	"github.com/saas-ph/api/internal/modules/announcements/application/usecases"
	"github.com/saas-ph/api/internal/modules/announcements/domain"
	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger            *slog.Logger
	AnnouncementsRepo domain.AnnouncementRepository
	AudiencesRepo     domain.AudienceRepository
	AcksRepo          domain.AckRepository
	// TxRunner es opcional. Cuando esta presente, CreateAnnouncement
	// abre una transaccion del Tenant DB para insertar el anuncio + sus
	// audiencias atomicamente. Cuando es nil (tests con repos en
	// memoria) la creacion ocurre sin transaccion.
	TxRunner usecases.TxRunner
	Now      func() time.Time
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

// --- Announcements ---

// createAnnouncement POST /announcements
func (h *handlers) createAnnouncement(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateAnnouncementRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	publishedBy := body.PublishedByUserID
	if publishedBy == "" {
		publishedBy = actorIDFromCtx(r)
	}
	pinned := false
	if body.Pinned != nil {
		pinned = *body.Pinned
	}
	auds := make([]usecases.AudienceInput, 0, len(body.Audiences))
	for _, a := range body.Audiences {
		auds = append(auds, usecases.AudienceInput{
			TargetType: a.TargetType,
			TargetID:   a.TargetID,
		})
	}

	uc := usecases.CreateAnnouncement{
		Announcements: h.deps.AnnouncementsRepo,
		Audiences:     h.deps.AudiencesRepo,
		TxRunner:      h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateAnnouncementInput{
		Title:             body.Title,
		Body:              body.Body,
		PublishedByUserID: publishedBy,
		Pinned:            pinned,
		ExpiresAt:         body.ExpiresAt,
		Audiences:         auds,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, announcementToDTO(out.Announcement, out.Audiences))
}

// getAnnouncement GET /announcements/{id}
func (h *handlers) getAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetAnnouncement{
		Announcements: h.deps.AnnouncementsRepo,
		Audiences:     h.deps.AudiencesRepo,
	}
	out, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, announcementToDTO(out.Announcement, out.Audiences))
}

// archiveAnnouncement DELETE /announcements/{id}
func (h *handlers) archiveAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.ArchiveAnnouncement{Announcements: h.deps.AnnouncementsRepo}
	if _, err := uc.Execute(r.Context(), usecases.ArchiveAnnouncementInput{
		ID:      id,
		ActorID: actorIDFromCtx(r),
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ackAnnouncement POST /announcements/{id}/ack
func (h *handlers) ackAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := actorIDFromCtx(r)
	uc := usecases.Acknowledge{Acks: h.deps.AcksRepo}
	out, err := uc.Execute(r.Context(), usecases.AcknowledgeInput{
		AnnouncementID: id,
		UserID:         userID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.AckResponse{
		ID:             out.ID,
		AnnouncementID: out.AnnouncementID,
		UserID:         out.UserID,
		AcknowledgedAt: out.AcknowledgedAt,
	})
}

// listFeed GET /announcements/feed?limit=&offset=
//
// TODO: en este MVP los scopes (role_ids, structure_ids, unit_ids) del
// usuario llegan vacios. Resolverlos requiere consultar los modulos
// authorization (roles asignados) y units (residencias del usuario), que
// hoy tienen repos stub. El usuario es el actor del contexto. Cuando los
// scopes estan vacios la consulta devuelve solo anuncios con
// audience='global', que es el comportamiento documentado.
func (h *handlers) listFeed(w http.ResponseWriter, r *http.Request) {
	userID := actorIDFromCtx(r)
	limit := parseInt32(r.URL.Query().Get("limit"), 20)
	offset := parseInt32(r.URL.Query().Get("offset"), 0)
	uc := usecases.ListFeed{Announcements: h.deps.AnnouncementsRepo}
	out, err := uc.Execute(r.Context(), usecases.FeedInput{
		UserID:       userID,
		RoleIDs:      nil,
		StructureIDs: nil,
		UnitIDs:      nil,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.FeedResponse{
		Items: make([]dto.AnnouncementResponse, 0, len(out.Items)),
		Total: out.Total,
	}
	for _, a := range out.Items {
		resp.Items = append(resp.Items, announcementToDTO(a, nil))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "announcements: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "announcements: unexpected error",
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

func parseInt32(s string, def int32) int32 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return def
	}
	return int32(v)
}

func announcementToDTO(a entities.Announcement, audiences []entities.Audience) dto.AnnouncementResponse {
	resp := dto.AnnouncementResponse{
		ID:          a.ID,
		Title:       a.Title,
		Body:        a.Body,
		PublishedBy: a.PublishedByUserID,
		PublishedAt: a.PublishedAt,
		Pinned:      a.Pinned,
		ExpiresAt:   a.ExpiresAt,
		Status:      string(a.Status),
		Audiences:   make([]dto.AudienceTarget, 0, len(audiences)),
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		Version:     a.Version,
	}
	for _, ad := range audiences {
		resp.Audiences = append(resp.Audiences, dto.AudienceTarget{
			TargetType: string(ad.TargetType),
			TargetID:   ad.TargetID,
		})
	}
	return resp
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

// actorIDFromCtx extrae el user_id del contexto. En MVP, el caller
// puede usar el header X-User-ID via un middleware externo o WithActorID.
func actorIDFromCtx(r *http.Request) string {
	if v, ok := r.Context().Value(actorCtxKey{}).(string); ok {
		return v
	}
	// Fallback en MVP: header X-User-ID.
	if v := r.Header.Get("X-User-ID"); v != "" {
		return v
	}
	return ""
}
