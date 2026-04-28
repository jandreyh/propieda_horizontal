// Package persistence implementa los repositorios del modulo
// authorization sobre PostgreSQL via pgx/sqlc.
//
// Diseno: stateless. El pool del tenant se resuelve por-request via
// `tenantctx.FromCtx`. Los metodos son stubs hasta que se complete la
// implementacion concreta (post-DoD MVP). Los tests del modulo usan
// mocks de las interfaces de dominio, asi que estos stubs no afectan
// la verificacion del DoD que cubre policies/usecases/middleware.
package persistence

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/authorization/domain"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

// RoleRepo implementa domain.RoleRepository (stub).
type RoleRepo struct{}

// NewRoleRepo construye un RoleRepo sin estado.
func NewRoleRepo() *RoleRepo { return &RoleRepo{} }

// PermissionRepo implementa domain.PermissionRepository (stub).
type PermissionRepo struct{}

// NewPermissionRepo construye un PermissionRepo sin estado.
func NewPermissionRepo() *PermissionRepo { return &PermissionRepo{} }

// AssignmentRepo implementa domain.AssignmentRepository (stub).
type AssignmentRepo struct{}

// NewAssignmentRepo construye un AssignmentRepo sin estado.
func NewAssignmentRepo() *AssignmentRepo { return &AssignmentRepo{} }

var errStub = errors.New("authorization persistence: implementacion concreta pendiente (post-DoD MVP)")

// ListActive stub.
func (r *RoleRepo) ListActive(ctx context.Context) ([]entities.Role, error) {
	_ = ctx
	return nil, errStub
}

// GetByID stub.
func (r *RoleRepo) GetByID(ctx context.Context, id string) (entities.Role, error) {
	_ = ctx
	_ = id
	return entities.Role{}, errStub
}

// GetByName stub.
func (r *RoleRepo) GetByName(ctx context.Context, name string) (entities.Role, error) {
	_ = ctx
	_ = name
	return entities.Role{}, errStub
}

// Create stub.
func (r *RoleRepo) Create(ctx context.Context, p domain.CreateRoleParams) (entities.Role, error) {
	_ = ctx
	_ = p
	return entities.Role{}, errStub
}

// UpdateName stub.
func (r *RoleRepo) UpdateName(ctx context.Context, p domain.UpdateRoleParams) (entities.Role, error) {
	_ = ctx
	_ = p
	return entities.Role{}, errStub
}

// Archive stub.
func (r *RoleRepo) Archive(ctx context.Context, id string, by *string) error {
	_ = ctx
	_ = id
	_ = by
	return errStub
}

// ListPermissionsForRole stub.
func (r *RoleRepo) ListPermissionsForRole(ctx context.Context, roleID string) ([]entities.Permission, error) {
	_ = ctx
	_ = roleID
	return nil, errStub
}

// AssignPermission stub.
func (r *RoleRepo) AssignPermission(ctx context.Context, roleID, permissionID string) error {
	_ = ctx
	_ = roleID
	_ = permissionID
	return errStub
}

// RevokePermission stub.
func (r *RoleRepo) RevokePermission(ctx context.Context, roleID, permissionID string) error {
	_ = ctx
	_ = roleID
	_ = permissionID
	return errStub
}

// ReplacePermissions stub.
func (r *RoleRepo) ReplacePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	_ = ctx
	_ = roleID
	_ = permissionIDs
	return errStub
}

// List stub.
func (r *PermissionRepo) List(ctx context.Context) ([]entities.Permission, error) {
	_ = ctx
	return nil, errStub
}

// GetByNamespace stub.
func (r *PermissionRepo) GetByNamespace(ctx context.Context, ns string) (entities.Permission, error) {
	_ = ctx
	_ = ns
	return entities.Permission{}, errStub
}

// Create stub.
func (r *AssignmentRepo) Create(ctx context.Context, p domain.AssignmentParams) (entities.RoleAssignment, error) {
	_ = ctx
	_ = p
	return entities.RoleAssignment{}, errStub
}

// Revoke stub.
func (r *AssignmentRepo) Revoke(ctx context.Context, id string, by *string, reason string) error {
	_ = ctx
	_ = id
	_ = by
	_ = reason
	return errStub
}

// GetActiveByUser stub.
func (r *AssignmentRepo) GetActiveByUser(ctx context.Context, userID string) ([]entities.RoleAssignment, error) {
	_ = ctx
	_ = userID
	return nil, errStub
}

// ListPermissionNamespacesForUser stub.
func (r *AssignmentRepo) ListPermissionNamespacesForUser(ctx context.Context, userID string) ([]string, error) {
	_ = ctx
	_ = userID
	return nil, errStub
}

var (
	_ domain.RoleRepository       = (*RoleRepo)(nil)
	_ domain.PermissionRepository = (*PermissionRepo)(nil)
	_ domain.AssignmentRepository = (*AssignmentRepo)(nil)
)
