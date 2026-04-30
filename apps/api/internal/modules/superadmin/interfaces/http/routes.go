// Package http del modulo superadmin expone los endpoints administrativos
// de plataforma. Vive fuera del tenant_resolver — la identidad del
// superadmin es global y la verificacion del rol se hace inline en el
// middleware de autorizacion.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/modules/provisioning"
	"github.com/saas-ph/api/internal/modules/superadmin/application/dto"
	"github.com/saas-ph/api/internal/modules/superadmin/application/usecases"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/middleware"
)

// SuperadminRole es el role marker que las claims del JWT deben llevar
// para acceder a estos endpoints.
const SuperadminRole = "platform_superadmin"

// Dependencies agrupa el cableado.
type Dependencies struct {
	Logger      *slog.Logger
	CentralPool *pgxpool.Pool
	Provisioner *provisioning.Provisioner
}

type handlers struct {
	logger          *slog.Logger
	createTenantUC  *usecases.CreateTenantUseCase
	listTenantsUC   *usecases.ListTenantsUseCase
}

// Mount registra los endpoints en r bajo /superadmin/. Aplica el
// requireSuperadmin middleware (ademas del PlatformAuth global del
// router padre).
func Mount(r chi.Router, deps Dependencies) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	h := &handlers{
		logger: logger,
		createTenantUC: usecases.NewCreateTenantUseCase(usecases.CreateTenantDeps{
			Provisioner: deps.Provisioner,
		}),
		listTenantsUC: usecases.NewListTenantsUseCase(usecases.ListTenantsDeps{
			CentralPool: deps.CentralPool,
		}),
	}
	r.Route("/superadmin", func(sr chi.Router) {
		sr.Use(requireSuperadmin)
		sr.Post("/tenants", h.createTenant)
		sr.Get("/tenants", h.listTenants)
	})
}

// requireSuperadmin chequea que las claims (puestas por PlatformAuth)
// tengan el rol platform_superadmin. PlatformAuth debe correr antes en
// la cadena de middlewares; aqui solo leemos el contexto.
func requireSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.PlatformAuthFromCtx(r.Context())
		if !ok || claims == nil {
			apperrors.Write(w, apperrors.Unauthorized("authentication required").WithInstance(r.URL.Path))
			return
		}
		if !slices.Contains(claims.Roles, SuperadminRole) {
			apperrors.Write(w, apperrors.Forbidden("superadmin role required").WithInstance(r.URL.Path))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *handlers) createTenant(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTenantRequest
	if err := decodeJSONBody(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	resp, err := h.createTenantUC.Execute(r.Context(), req)
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *handlers) listTenants(w http.ResponseWriter, r *http.Request) {
	resp, err := h.listTenantsUC.Execute(r.Context())
	if err != nil {
		h.writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) writeUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecases.ErrInvalidInput):
		apperrors.Write(w, apperrors.BadRequest("invalid input").WithInstance(r.URL.Path))
	default:
		if h.logger != nil {
			h.logger.ErrorContext(r.Context(), "superadmin: internal error",
				slog.String("error", err.Error()),
				slog.String("path", r.URL.Path),
			)
		}
		apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
	}
}

func decodeJSONBody(r *http.Request, dst any) error {
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
