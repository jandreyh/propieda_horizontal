package entities

import "time"

// Constantes para scope_type del modelo RBAC. Solo MVP.
const (
	ScopeTenant = "tenant"
	ScopeTower  = "tower"
	ScopeUnit   = "unit"
)

// RoleAssignment representa una asignacion (user, role, scope?) con su
// historico de revocacion. NO usa soft delete tradicional: las
// revocaciones se materializan via revoked_at + revocation_reason.
type RoleAssignment struct {
	ID               string
	UserID           string
	RoleID           string
	ScopeType        *string
	ScopeID          *string
	GrantedBy        *string
	GrantedAt        time.Time
	RevokedAt        *time.Time
	RevocationReason *string
	Status           string

	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy *string
	UpdatedBy *string
	Version   int
}

// IsActive indica si la asignacion esta vigente (no revocada).
func (a RoleAssignment) IsActive() bool {
	return a.RevokedAt == nil && a.Status == StatusActive
}
