package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/authorization/application/usecases"
	"github.com/saas-ph/api/internal/modules/authorization/domain"
)

// Dependencies agrupa lo que el modulo necesita para arrancar.
//
// Reglas de cableado:
//   - Logger es opcional; si es nil se silencian errores 500.
//   - RoleRepo, PermissionRepo, AssignmentRepo son obligatorios — son
//     inyectados por main.go con la implementacion concreta sobre el
//     pool del Tenant DB resuelto por TenantResolver.
//   - Now es opcional; si es nil se usa time.Now (el cache de
//     permisos en memoria depende de este reloj).
type Dependencies struct {
	Logger         *slog.Logger
	RoleRepo       domain.RoleRepository
	PermissionRepo domain.PermissionRepository
	AssignmentRepo domain.AssignmentRepository
	Now            func() time.Time
}

// Mount monta las rutas del modulo authorization en `r`. Aplica el
// middleware RequirePermission a cada endpoint segun la matriz de
// permisos definida en el ADR 0003 / spec del modulo.
//
// IMPORTANTE: este modulo asume que un middleware de autenticacion
// previo ya inyecto user_id (y opcionalmente session_id) en el contexto
// via WithUserID/WithSessionID. La presencia del tenant pool en
// contexto se asume garantizada por TenantResolver aguas arriba.
func Mount(r chi.Router, deps Dependencies) {
	if deps.Now == nil {
		deps.Now = time.Now
	}

	resolveUC := usecases.ResolveUserPermissions{Assignments: deps.AssignmentRepo}
	cache := NewPermissionCache(60*time.Second, deps.Now)
	mw := MiddlewareConfig{
		Resolver: resolverAdapter{uc: resolveUC},
		Cache:    cache,
	}

	h := Handlers{
		Logger:       deps.Logger,
		List:         usecases.ListRoles{Roles: deps.RoleRepo},
		Create:       usecases.CreateRole{Roles: deps.RoleRepo},
		Get:          usecases.GetRole{Roles: deps.RoleRepo},
		Update:       usecases.UpdateRole{Roles: deps.RoleRepo},
		Delete:       usecases.DeleteRole{Roles: deps.RoleRepo},
		ListPerms:    usecases.ListPermissions{Permissions: deps.PermissionRepo},
		Assign:       usecases.AssignRole{Roles: deps.RoleRepo, Assignments: deps.AssignmentRepo},
		Unassign:     usecases.UnassignRole{Assignments: deps.AssignmentRepo},
		ResolvePerms: resolveUC,
	}

	r.Route("/roles", func(rr chi.Router) {
		rr.With(mw.RequirePermission("role.read")).Get("/", h.ListRoles)
		rr.With(mw.RequirePermission("role.create")).Post("/", h.CreateRole)
		rr.With(mw.RequirePermission("role.read")).Get("/{id}", h.GetRole)
		rr.With(mw.RequirePermission("role.update")).Put("/{id}", h.UpdateRole)
		rr.With(mw.RequirePermission("role.delete")).Delete("/{id}", h.DeleteRole)
	})

	r.With(mw.RequirePermission("permission.read")).Get("/permissions", h.ListPermissions)

	r.Route("/users/{id}/roles", func(rr chi.Router) {
		rr.With(mw.RequirePermission("user.assign_role")).Post("/", h.AssignRole)
		rr.With(mw.RequirePermission("user.unassign_role")).Delete("/{role_id}", h.UnassignRole)
	})

	// /users/{id}/permissions: el actor puede leer si tiene
	// `user.assign_role` (admins) o si esta consultando sus propios
	// permisos. El segundo caso se permite sin permiso explicito; el
	// middleware solo exige `user.assign_role` y el handler agrega el
	// short-circuit antes de invocarlo via wrapper.
	r.Get("/users/{id}/permissions", selfOrPermissionMiddleware(mw, "user.assign_role")(h.UserPermissions).ServeHTTP)
}

// selfOrPermissionMiddleware permite el paso si el usuario actor es el
// mismo que el path param `id` (auto-consulta) o si tiene `ns`. Aplica a
// GET /users/:id/permissions.
func selfOrPermissionMiddleware(mw MiddlewareConfig, ns string) func(http.HandlerFunc) http.Handler {
	return func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := UserIDFromCtx(r.Context())
			target := chi.URLParam(r, "id")
			if actor != "" && actor == target {
				next.ServeHTTP(w, r)
				return
			}
			mw.RequirePermission(ns)(next).ServeHTTP(w, r)
		})
	}
}

// resolverAdapter adapta el usecase ResolveUserPermissions a la
// interfaz permissionResolver consumida por el middleware.
type resolverAdapter struct {
	uc usecases.ResolveUserPermissions
}

// Permissions implementa permissionResolver.
func (a resolverAdapter) Permissions(ctx context.Context, userID string) ([]string, error) {
	out, err := a.uc.Execute(ctx, userID)
	if err != nil {
		return nil, err
	}
	return out.Permissions, nil
}
