// Package domain define las interfaces de repositorio que los usecases
// de authorization consumen. Las implementaciones concretas viven en
// `infrastructure/persistence`.
package domain

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

// Errores estandar del dominio. Los handlers HTTP los mapean a
// Problem+JSON (404, 409, etc).
var (
	// ErrRoleNotFound se devuelve cuando un repositorio no encuentra el rol.
	ErrRoleNotFound = errors.New("authorization: role not found")
	// ErrPermissionNotFound se devuelve cuando no existe el permiso.
	ErrPermissionNotFound = errors.New("authorization: permission not found")
	// ErrAssignmentNotFound se devuelve cuando no existe la asignacion.
	ErrAssignmentNotFound = errors.New("authorization: assignment not found")
	// ErrRoleNameTaken se devuelve cuando se intenta crear/renombrar un
	// rol con un nombre ya existente.
	ErrRoleNameTaken = errors.New("authorization: role name already taken")
	// ErrSystemRoleImmutable se devuelve al intentar editar/borrar un rol
	// is_system=true desde la API de gestion.
	ErrSystemRoleImmutable = errors.New("authorization: system roles are immutable")
	// ErrAssignmentDuplicate se devuelve al asignar un rol al mismo
	// usuario con el mismo scope dos veces (mientras este activa).
	ErrAssignmentDuplicate = errors.New("authorization: assignment already exists")
)

// CreateRoleParams agrupa los datos para crear un rol custom.
type CreateRoleParams struct {
	Name        string
	Description string
	CreatedBy   *string
}

// UpdateRoleParams agrupa datos para renombrar/redescribir un rol.
type UpdateRoleParams struct {
	ID          string
	Name        string
	Description string
	UpdatedBy   *string
	Version     int
}

// AssignmentParams agrupa los datos para crear una asignacion.
type AssignmentParams struct {
	UserID    string
	RoleID    string
	ScopeType *string
	ScopeID   *string
	GrantedBy *string
}

// RoleRepository abstrae el acceso a la tabla `roles` y `role_permissions`.
type RoleRepository interface {
	ListActive(ctx context.Context) ([]entities.Role, error)
	GetByID(ctx context.Context, id string) (entities.Role, error)
	GetByName(ctx context.Context, name string) (entities.Role, error)
	Create(ctx context.Context, p CreateRoleParams) (entities.Role, error)
	UpdateName(ctx context.Context, p UpdateRoleParams) (entities.Role, error)
	Archive(ctx context.Context, id string, by *string) error

	ListPermissionsForRole(ctx context.Context, roleID string) ([]entities.Permission, error)
	AssignPermission(ctx context.Context, roleID, permissionID string) error
	RevokePermission(ctx context.Context, roleID, permissionID string) error
	ReplacePermissions(ctx context.Context, roleID string, permissionIDs []string) error
}

// PermissionRepository abstrae el catalogo estatico de permisos.
type PermissionRepository interface {
	List(ctx context.Context) ([]entities.Permission, error)
	GetByNamespace(ctx context.Context, ns string) (entities.Permission, error)
}

// AssignmentRepository abstrae user_role_assignments.
type AssignmentRepository interface {
	Create(ctx context.Context, p AssignmentParams) (entities.RoleAssignment, error)
	Revoke(ctx context.Context, id string, by *string, reason string) error
	GetActiveByUser(ctx context.Context, userID string) ([]entities.RoleAssignment, error)
	// ListPermissionNamespacesForUser hace el JOIN cross
	// users -> assignments -> roles -> permissions y devuelve los
	// namespaces unicos de los permisos efectivos del usuario, respetando
	// soft delete y revocaciones.
	ListPermissionNamespacesForUser(ctx context.Context, userID string) ([]string, error)
}
