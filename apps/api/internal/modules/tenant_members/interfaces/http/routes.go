// Package http del modulo tenant_members. Estos endpoints VIVEN bajo
// tenant_resolver — el tenant ya esta en el contexto cuando entran.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/tenant_members/application/dto"
	"github.com/saas-ph/api/internal/modules/tenant_members/application/usecases"
	"github.com/saas-ph/api/internal/modules/tenant_members/domain"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa el cableado.
type Dependencies struct {
	Logger   *slog.Logger
	Links    domain.LinkRepository
	Enricher domain.EnricherRepository
}

type handlers struct {
	logger   *slog.Logger
	addUC    *usecases.AddByCodeUseCase
	listUC   *usecases.ListUseCase
	updateUC *usecases.UpdateUseCase
	blockUC  *usecases.BlockUseCase
}

// Mount registra los endpoints en r.
//
// Endpoints:
//   - POST   /tenant-members
//   - GET    /tenant-members
//   - PUT    /tenant-members/{id}
//   - POST   /tenant-members/{id}/block
func Mount(r chi.Router, deps Dependencies) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	h := &handlers{
		logger: logger,
		addUC: usecases.NewAddByCodeUseCase(usecases.AddByCodeDeps{
			Links: deps.Links, Enricher: deps.Enricher,
		}),
		listUC: usecases.NewListUseCase(usecases.ListDeps{
			Links: deps.Links, Enricher: deps.Enricher,
		}),
		updateUC: usecases.NewUpdateUseCase(usecases.UpdateDeps{
			Links: deps.Links, Enricher: deps.Enricher,
		}),
		blockUC: usecases.NewBlockUseCase(usecases.BlockDeps{
			Links:    deps.Links,
			Enricher: deps.Enricher,
		}),
	}
	r.Route("/tenant-members", func(tr chi.Router) {
		tr.Post("/", h.add)
		tr.Get("/", h.list)
		tr.Put("/{id}", h.update)
		tr.Post("/{id}/block", h.block)
	})
}

func (h *handlers) add(w http.ResponseWriter, r *http.Request) {
	var req dto.AddMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	resp, err := h.addUC.Execute(r.Context(), req)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	resp, err := h.listUC.Execute(r.Context())
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	resp, err := h.updateUC.Execute(r.Context(), id, req)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) block(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.blockUC.Execute(r.Context(), id); err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) writeUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecases.ErrInvalidInput):
		apperrors.Write(w, apperrors.BadRequest("invalid input").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrCodeNotFound):
		apperrors.Write(w, apperrors.NotFound("public_code not found").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrAlreadyLinked):
		apperrors.Write(w, apperrors.Conflict("user already linked").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrLinkNotFound):
		apperrors.Write(w, apperrors.NotFound("link not found").WithInstance(r.URL.Path))
	case errors.Is(err, usecases.ErrVersionMismatch):
		apperrors.Write(w, apperrors.Conflict("version mismatch").WithInstance(r.URL.Path))
	default:
		if h.logger != nil {
			h.logger.ErrorContext(r.Context(), "tenant_members: internal",
				slog.String("error", err.Error()),
				slog.String("path", r.URL.Path),
			)
		}
		apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
	}
}

func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid json body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
