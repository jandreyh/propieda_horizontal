package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/authorization/application/dto"
	"github.com/saas-ph/api/internal/modules/authorization/application/usecases"
	"github.com/saas-ph/api/internal/modules/authorization/domain"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Handlers agrupa todos los handlers HTTP del modulo authorization.
type Handlers struct {
	Logger *slog.Logger

	List         usecases.ListRoles
	Create       usecases.CreateRole
	Get          usecases.GetRole
	Update       usecases.UpdateRole
	Delete       usecases.DeleteRole
	ListPerms    usecases.ListPermissions
	Assign       usecases.AssignRole
	Unassign     usecases.UnassignRole
	ResolvePerms usecases.ResolveUserPermissions
}

// ListRoles GET /roles
func (h Handlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	out, err := h.List.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// CreateRole POST /roles
func (h Handlers) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	actor := userPtr(UserIDFromCtx(r.Context()))
	out, err := h.Create.Execute(r.Context(), usecases.CreateRoleInput{
		Name:          req.Name,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
		ActorUserID:   actor,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// GetRole GET /roles/:id
func (h Handlers) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := h.Get.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// UpdateRole PUT /roles/:id
func (h Handlers) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	actor := userPtr(UserIDFromCtx(r.Context()))
	in := usecases.UpdateRoleInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		ActorUserID: actor,
	}
	if req.PermissionIDs != nil {
		ids := req.PermissionIDs
		in.PermissionIDs = &ids
	}
	out, err := h.Update.Execute(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DeleteRole DELETE /roles/:id
func (h Handlers) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actor := userPtr(UserIDFromCtx(r.Context()))
	if err := h.Delete.Execute(r.Context(), usecases.DeleteRoleInput{ID: id, ActorUserID: actor}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListPermissions GET /permissions
func (h Handlers) ListPermissions(w http.ResponseWriter, r *http.Request) {
	out, err := h.ListPerms.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// AssignRole POST /users/:id/roles
func (h Handlers) AssignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	var req dto.AssignRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	actor := userPtr(UserIDFromCtx(r.Context()))
	out, err := h.Assign.Execute(r.Context(), usecases.AssignRoleInput{
		UserID:      userID,
		RoleID:      req.RoleID,
		ScopeType:   req.ScopeType,
		ScopeID:     req.ScopeID,
		GrantedBy:   actor,
		ActorUserID: actor,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// UnassignRole DELETE /users/:id/roles/:role_id
func (h Handlers) UnassignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	roleID := chi.URLParam(r, "role_id")
	var req dto.UnassignRoleRequest
	// El body es opcional; si esta vacio o invalido, seguimos con reason="".
	if r.ContentLength > 0 {
		_ = decodeJSON(r, &req)
	}
	actor := userPtr(UserIDFromCtx(r.Context()))
	if err := h.Unassign.Execute(r.Context(), usecases.UnassignRoleInput{
		UserID:      userID,
		RoleID:      roleID,
		Reason:      req.Reason,
		ActorUserID: actor,
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UserPermissions GET /users/:id/permissions
func (h Handlers) UserPermissions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	out, err := h.ResolvePerms.Execute(r.Context(), userID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// fail mapea errores de dominio a Problem+JSON.
func (h Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrRoleNotFound),
		errors.Is(err, domain.ErrPermissionNotFound),
		errors.Is(err, domain.ErrAssignmentNotFound):
		apperrors.Write(w, apperrors.NotFound(err.Error()).WithInstance(r.URL.Path))
		return
	case errors.Is(err, domain.ErrRoleNameTaken),
		errors.Is(err, domain.ErrAssignmentDuplicate):
		apperrors.Write(w, apperrors.Conflict(err.Error()).WithInstance(r.URL.Path))
		return
	case errors.Is(err, domain.ErrSystemRoleImmutable):
		apperrors.Write(w, apperrors.Forbidden(err.Error()).WithInstance(r.URL.Path))
		return
	}
	if strings.Contains(err.Error(), "is required") || strings.Contains(err.Error(), "scope") {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	if h.Logger != nil {
		h.Logger.ErrorContext(r.Context(), "authorization handler error",
			slog.String("error", err.Error()))
	}
	apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func userPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
