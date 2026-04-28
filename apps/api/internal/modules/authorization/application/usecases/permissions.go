package usecases

import (
	"context"

	"github.com/saas-ph/api/internal/modules/authorization/application/dto"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

// ListPermissions devuelve el catalogo estatico de permisos.
type ListPermissions struct {
	Permissions interface {
		List(ctx context.Context) ([]entities.Permission, error)
	}
}

// Execute corre el caso de uso.
func (uc ListPermissions) Execute(ctx context.Context) ([]dto.PermissionDTO, error) {
	perms, err := uc.Permissions.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.PermissionDTO, 0, len(perms))
	for _, p := range perms {
		out = append(out, permToDTO(p))
	}
	return out, nil
}
