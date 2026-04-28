package usecases

import (
	"context"
	"errors"
	"strings"

	"github.com/saas-ph/api/internal/modules/authorization/application/dto"
	"github.com/saas-ph/api/internal/modules/authorization/domain"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

// AssignRoleInput agrupa parametros del usecase AssignRole.
type AssignRoleInput struct {
	UserID      string
	RoleID      string
	ScopeType   *string
	ScopeID     *string
	GrantedBy   *string
	ActorUserID *string
}

// AssignRole crea una asignacion (user, role, scope?). Valida la
// coherencia (scope_type, scope_id) segun ADR 0003:
//   - sin scope: ambos nil.
//   - scope tenant: scope_type='tenant' y scope_id=nil.
//   - scope tower/unit: ambos no nil.
type AssignRole struct {
	Roles interface {
		GetByID(ctx context.Context, id string) (entities.Role, error)
	}
	Assignments interface {
		Create(ctx context.Context, p domain.AssignmentParams) (entities.RoleAssignment, error)
	}
}

// Execute corre el caso de uso.
func (uc AssignRole) Execute(ctx context.Context, in AssignRoleInput) (dto.AssignmentDTO, error) {
	if strings.TrimSpace(in.UserID) == "" {
		return dto.AssignmentDTO{}, errors.New("authorization: user_id is required")
	}
	if strings.TrimSpace(in.RoleID) == "" {
		return dto.AssignmentDTO{}, errors.New("authorization: role_id is required")
	}

	if err := validateScope(in.ScopeType, in.ScopeID); err != nil {
		return dto.AssignmentDTO{}, err
	}

	if _, err := uc.Roles.GetByID(ctx, in.RoleID); err != nil {
		return dto.AssignmentDTO{}, err
	}

	a, err := uc.Assignments.Create(ctx, domain.AssignmentParams{
		UserID:    in.UserID,
		RoleID:    in.RoleID,
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
		GrantedBy: in.GrantedBy,
	})
	if err != nil {
		return dto.AssignmentDTO{}, err
	}
	return assignmentToDTO(a), nil
}

// UnassignRoleInput agrupa parametros del usecase UnassignRole.
type UnassignRoleInput struct {
	UserID      string
	RoleID      string
	Reason      string
	ActorUserID *string
}

// UnassignRole busca la asignacion activa (UserID, RoleID) y la revoca.
type UnassignRole struct {
	Assignments interface {
		GetActiveByUser(ctx context.Context, userID string) ([]entities.RoleAssignment, error)
		Revoke(ctx context.Context, id string, by *string, reason string) error
	}
}

// Execute corre el caso de uso.
func (uc UnassignRole) Execute(ctx context.Context, in UnassignRoleInput) error {
	if strings.TrimSpace(in.UserID) == "" {
		return errors.New("authorization: user_id is required")
	}
	if strings.TrimSpace(in.RoleID) == "" {
		return errors.New("authorization: role_id is required")
	}

	active, err := uc.Assignments.GetActiveByUser(ctx, in.UserID)
	if err != nil {
		return err
	}
	for _, a := range active {
		if a.RoleID == in.RoleID {
			return uc.Assignments.Revoke(ctx, a.ID, in.ActorUserID, in.Reason)
		}
	}
	return domain.ErrAssignmentNotFound
}

// ResolveUserPermissions devuelve los namespaces de permisos efectivos
// para un usuario (union de todas sus asignaciones activas).
type ResolveUserPermissions struct {
	Assignments interface {
		ListPermissionNamespacesForUser(ctx context.Context, userID string) ([]string, error)
	}
}

// Execute corre el caso de uso.
func (uc ResolveUserPermissions) Execute(ctx context.Context, userID string) (dto.EffectivePermissionsResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return dto.EffectivePermissionsResponse{}, errors.New("authorization: user_id is required")
	}
	ns, err := uc.Assignments.ListPermissionNamespacesForUser(ctx, userID)
	if err != nil {
		return dto.EffectivePermissionsResponse{}, err
	}
	if ns == nil {
		ns = []string{}
	}
	return dto.EffectivePermissionsResponse{
		UserID:      userID,
		Permissions: ns,
	}, nil
}

// validateScope aplica las reglas del CHECK constraint a nivel de
// dominio (defensa en profundidad).
func validateScope(scopeType, scopeID *string) error {
	if scopeType == nil && scopeID == nil {
		return nil
	}
	if scopeType == nil && scopeID != nil {
		return errors.New("authorization: scope_id provided without scope_type")
	}
	switch *scopeType {
	case entities.ScopeTenant:
		if scopeID != nil {
			return errors.New("authorization: scope_id must be null for scope_type='tenant'")
		}
		return nil
	case entities.ScopeTower, entities.ScopeUnit:
		if scopeID == nil || strings.TrimSpace(*scopeID) == "" {
			return errors.New("authorization: scope_id is required for scope_type='tower'|'unit'")
		}
		return nil
	default:
		return errors.New("authorization: invalid scope_type")
	}
}

func assignmentToDTO(a entities.RoleAssignment) dto.AssignmentDTO {
	return dto.AssignmentDTO{
		ID:               a.ID,
		UserID:           a.UserID,
		RoleID:           a.RoleID,
		ScopeType:        a.ScopeType,
		ScopeID:          a.ScopeID,
		GrantedBy:        a.GrantedBy,
		GrantedAt:        a.GrantedAt,
		RevokedAt:        a.RevokedAt,
		RevocationReason: a.RevocationReason,
		Status:           a.Status,
	}
}
