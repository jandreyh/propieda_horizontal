// Package usecases contiene la orquestacion del modulo authorization.
// Los usecases reciben interfaces (RoleRepository, etc.) por DI y NO
// conocen HTTP ni DB.
package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/authorization/application/dto"
	"github.com/saas-ph/api/internal/modules/authorization/domain"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

// CreateRoleInput agrupa parametros del usecase CreateRole.
type CreateRoleInput struct {
	Name          string
	Description   string
	PermissionIDs []string
	ActorUserID   *string
}

// CreateRole crea un rol custom (is_system=false implicito) con la lista
// inicial de permisos. Errores tipicos:
//   - dominio: ErrRoleNameTaken si el nombre ya existe.
//   - dominio: ErrPermissionNotFound si algun permission_id no existe.
type CreateRole struct {
	Roles RoleSetter
}

// RoleSetter es el subset de RoleRepository que CreateRole / UpdateRole
// consumen. Es la superficie minima necesaria para escribir roles +
// reemplazar sus permisos.
type RoleSetter interface {
	GetByName(ctx context.Context, name string) (entities.Role, error)
	GetByID(ctx context.Context, id string) (entities.Role, error)
	Create(ctx context.Context, p domain.CreateRoleParams) (entities.Role, error)
	UpdateName(ctx context.Context, p domain.UpdateRoleParams) (entities.Role, error)
	ReplacePermissions(ctx context.Context, roleID string, permissionIDs []string) error
	ListPermissionsForRole(ctx context.Context, roleID string) ([]entities.Permission, error)
}

// Execute corre el caso de uso.
func (uc CreateRole) Execute(ctx context.Context, in CreateRoleInput) (dto.RoleDTO, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return dto.RoleDTO{}, errors.New("authorization: name is required")
	}

	existing, err := uc.Roles.GetByName(ctx, name)
	if err == nil && existing.ID != "" {
		return dto.RoleDTO{}, domain.ErrRoleNameTaken
	}
	if err != nil && !errors.Is(err, domain.ErrRoleNotFound) {
		return dto.RoleDTO{}, err
	}

	role, err := uc.Roles.Create(ctx, domain.CreateRoleParams{
		Name:        name,
		Description: in.Description,
		CreatedBy:   in.ActorUserID,
	})
	if err != nil {
		return dto.RoleDTO{}, err
	}

	if len(in.PermissionIDs) > 0 {
		if err := uc.Roles.ReplacePermissions(ctx, role.ID, in.PermissionIDs); err != nil {
			return dto.RoleDTO{}, err
		}
	}

	perms, err := uc.Roles.ListPermissionsForRole(ctx, role.ID)
	if err != nil {
		return dto.RoleDTO{}, err
	}
	role.Permissions = perms

	return roleToDTO(role), nil
}

// UpdateRoleInput agrupa parametros del usecase UpdateRole.
type UpdateRoleInput struct {
	ID            string
	Name          string
	Description   string
	PermissionIDs *[]string // nil = no tocar permisos
	Version       int
	ActorUserID   *string
}

// UpdateRole renombra/redescribe y opcionalmente reemplaza los permisos
// asignados a un rol. PROHIBIDO sobre roles is_system=true.
type UpdateRole struct {
	Roles RoleSetter
}

// Execute corre el caso de uso.
func (uc UpdateRole) Execute(ctx context.Context, in UpdateRoleInput) (dto.RoleDTO, error) {
	role, err := uc.Roles.GetByID(ctx, in.ID)
	if err != nil {
		return dto.RoleDTO{}, err
	}
	if role.IsSystem {
		return dto.RoleDTO{}, domain.ErrSystemRoleImmutable
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return dto.RoleDTO{}, errors.New("authorization: name is required")
	}

	// Si el nombre cambio, validar que no choque con otro rol.
	if !strings.EqualFold(name, role.Name) {
		if other, lookupErr := uc.Roles.GetByName(ctx, name); lookupErr == nil && other.ID != "" && other.ID != role.ID {
			return dto.RoleDTO{}, domain.ErrRoleNameTaken
		} else if lookupErr != nil && !errors.Is(lookupErr, domain.ErrRoleNotFound) {
			return dto.RoleDTO{}, lookupErr
		}
	}

	updated, err := uc.Roles.UpdateName(ctx, domain.UpdateRoleParams{
		ID:          role.ID,
		Name:        name,
		Description: in.Description,
		UpdatedBy:   in.ActorUserID,
		Version:     in.Version,
	})
	if err != nil {
		return dto.RoleDTO{}, err
	}

	if in.PermissionIDs != nil {
		if err := uc.Roles.ReplacePermissions(ctx, updated.ID, *in.PermissionIDs); err != nil {
			return dto.RoleDTO{}, err
		}
	}

	perms, err := uc.Roles.ListPermissionsForRole(ctx, updated.ID)
	if err != nil {
		return dto.RoleDTO{}, err
	}
	updated.Permissions = perms

	return roleToDTO(updated), nil
}

// DeleteRole archiva (soft delete) un rol custom. NO se permite sobre
// roles is_system=true.
type DeleteRole struct {
	Roles interface {
		GetByID(ctx context.Context, id string) (entities.Role, error)
		Archive(ctx context.Context, id string, by *string) error
	}
}

// DeleteRoleInput agrupa parametros del usecase DeleteRole.
type DeleteRoleInput struct {
	ID          string
	ActorUserID *string
}

// Execute corre el caso de uso.
func (uc DeleteRole) Execute(ctx context.Context, in DeleteRoleInput) error {
	role, err := uc.Roles.GetByID(ctx, in.ID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return domain.ErrSystemRoleImmutable
	}
	return uc.Roles.Archive(ctx, role.ID, in.ActorUserID)
}

// ListRoles devuelve los roles activos.
type ListRoles struct {
	Roles interface {
		ListActive(ctx context.Context) ([]entities.Role, error)
	}
}

// Execute corre el caso de uso.
func (uc ListRoles) Execute(ctx context.Context) ([]dto.RoleDTO, error) {
	roles, err := uc.Roles.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.RoleDTO, 0, len(roles))
	for _, r := range roles {
		out = append(out, roleToDTO(r))
	}
	return out, nil
}

// GetRole devuelve un rol con sus permisos.
type GetRole struct {
	Roles interface {
		GetByID(ctx context.Context, id string) (entities.Role, error)
		ListPermissionsForRole(ctx context.Context, roleID string) ([]entities.Permission, error)
	}
}

// Execute corre el caso de uso.
func (uc GetRole) Execute(ctx context.Context, id string) (dto.RoleDTO, error) {
	role, err := uc.Roles.GetByID(ctx, id)
	if err != nil {
		return dto.RoleDTO{}, err
	}
	perms, err := uc.Roles.ListPermissionsForRole(ctx, role.ID)
	if err != nil {
		return dto.RoleDTO{}, err
	}
	role.Permissions = perms
	return roleToDTO(role), nil
}

func roleToDTO(r entities.Role) dto.RoleDTO {
	out := dto.RoleDTO{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Status:      r.Status,
		Version:     r.Version,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if len(r.Permissions) > 0 {
		out.Permissions = make([]dto.PermissionDTO, 0, len(r.Permissions))
		for _, p := range r.Permissions {
			out.Permissions = append(out.Permissions, permToDTO(p))
		}
	}
	return out
}

func permToDTO(p entities.Permission) dto.PermissionDTO {
	return dto.PermissionDTO{
		ID:          p.ID,
		Namespace:   p.Namespace,
		Description: p.Description,
		Status:      p.Status,
	}
}

// NowFn es un alias usado por tests para inyectar reloj fijo.
type NowFn = func() time.Time
