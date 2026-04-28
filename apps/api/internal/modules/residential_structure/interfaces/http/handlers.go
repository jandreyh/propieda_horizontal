// Package http contiene los adaptadores HTTP del modulo
// residential_structure.
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
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/residential_structure/application/dto"
	"github.com/saas-ph/api/internal/modules/residential_structure/application/usecases"
	"github.com/saas-ph/api/internal/modules/residential_structure/domain"
	"github.com/saas-ph/api/internal/modules/residential_structure/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye el repo y lo inyecta aqui.
type Dependencies struct {
	Logger        *slog.Logger
	StructureRepo domain.StructureRepository
	Now           func() time.Time
}

// validate completa los defaults razonables (slogger, clock).
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

// listStructures GET /structures
func (h *handlers) listStructures(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListStructures{Repo: h.deps.StructureRepo}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListStructuresResponse{
		Items: make([]dto.StructureResponse, 0, len(out.Items)),
		Total: out.Total,
	}
	for _, s := range out.Items {
		resp.Items = append(resp.Items, structureToDTO(s))
	}
	writeJSON(w, http.StatusOK, resp)
}

// getStructure GET /structures/{id}
func (h *handlers) getStructure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetStructure{Repo: h.deps.StructureRepo}
	s, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, structureToDTO(s))
}

// createStructure POST /structures
func (h *handlers) createStructure(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateStructureRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.CreateStructure{Repo: h.deps.StructureRepo}
	s, err := uc.Execute(r.Context(), usecases.CreateStructureInput{
		Name:        body.Name,
		Type:        body.Type,
		ParentID:    body.ParentID,
		Description: body.Description,
		OrderIndex:  body.OrderIndex,
		ActorID:     actorIDFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, structureToDTO(s))
}

// updateStructure PUT /structures/{id}
func (h *handlers) updateStructure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.UpdateStructureRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.UpdateStructure{Repo: h.deps.StructureRepo}
	s, err := uc.Execute(r.Context(), usecases.UpdateStructureInput{
		ID:              id,
		Name:            body.Name,
		Type:            body.Type,
		ParentID:        body.ParentID,
		Description:     body.Description,
		OrderIndex:      body.OrderIndex,
		ActorID:         actorIDFromCtx(r),
		ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, structureToDTO(s))
}

// deleteStructure DELETE /structures/{id}
func (h *handlers) deleteStructure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.ArchiveStructure{Repo: h.deps.StructureRepo}
	if err := uc.Execute(r.Context(), id, actorIDFromCtx(r)); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "residential_structure: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "residential_structure: unexpected error",
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

func structureToDTO(s entities.Structure) dto.StructureResponse {
	return dto.StructureResponse{
		ID:          s.ID,
		Name:        s.Name,
		Type:        string(s.Type),
		ParentID:    s.ParentID,
		Description: s.Description,
		OrderIndex:  s.OrderIndex,
		Status:      string(s.Status),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Version:     s.Version,
	}
}

// actorCtxKey aisla la clave de contexto para el actor (user_id) que un
// middleware externo inyecta. El modulo no importa el modulo de auth.
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
