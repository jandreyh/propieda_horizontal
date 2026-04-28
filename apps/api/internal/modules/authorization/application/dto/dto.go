// Package dto contiene los Data Transfer Objects de authorization. Los
// tags JSON viven AQUI (no en domain/entities).
package dto

import "time"

// PermissionDTO es la representacion serializable de un permiso.
type PermissionDTO struct {
	ID          string `json:"id"`
	Namespace   string `json:"namespace"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
}

// RoleDTO es la representacion serializable de un rol con su lista
// (opcional) de permisos.
type RoleDTO struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	IsSystem    bool            `json:"is_system"`
	Status      string          `json:"status"`
	Version     int             `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Permissions []PermissionDTO `json:"permissions,omitempty"`
}

// AssignmentDTO es la representacion serializable de una asignacion.
type AssignmentDTO struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	RoleID           string     `json:"role_id"`
	ScopeType        *string    `json:"scope_type,omitempty"`
	ScopeID          *string    `json:"scope_id,omitempty"`
	GrantedBy        *string    `json:"granted_by,omitempty"`
	GrantedAt        time.Time  `json:"granted_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason *string    `json:"revocation_reason,omitempty"`
	Status           string     `json:"status"`
}

// CreateRoleRequest es el payload de POST /roles.
type CreateRoleRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	PermissionIDs []string `json:"permission_ids,omitempty"`
}

// UpdateRoleRequest es el payload de PUT /roles/:id.
type UpdateRoleRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	PermissionIDs []string `json:"permission_ids,omitempty"`
	Version       int      `json:"version"`
}

// AssignRoleRequest es el payload de POST /users/:id/roles.
type AssignRoleRequest struct {
	RoleID    string  `json:"role_id"`
	ScopeType *string `json:"scope_type,omitempty"`
	ScopeID   *string `json:"scope_id,omitempty"`
}

// UnassignRoleRequest es el payload (opcional) de DELETE /users/:id/roles/:role_id.
type UnassignRoleRequest struct {
	Reason string `json:"reason,omitempty"`
}

// EffectivePermissionsResponse es la respuesta de GET /users/:id/permissions.
type EffectivePermissionsResponse struct {
	UserID      string   `json:"user_id"`
	Permissions []string `json:"permissions"`
}
