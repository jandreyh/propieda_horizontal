// Package usecases orquesta la logica de aplicacion del modulo
// residential_structure. Cada usecase recibe sus dependencias por
// inyeccion (interfaces de dominio) y NO conoce HTTP ni la base.
package usecases

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/saas-ph/api/internal/platform/errors"

	"github.com/saas-ph/api/internal/modules/residential_structure/domain"
	"github.com/saas-ph/api/internal/modules/residential_structure/domain/entities"
)

// nameMaxLen es el limite razonable para nombres de torre/bloque/etapa.
const nameMaxLen = 200

// ListResponse agrupa la salida del listado.
type ListResponse struct {
	Items []entities.Structure
	Total int
}

// ListStructures lista todas las estructuras activas.
type ListStructures struct {
	Repo domain.StructureRepository
}

// Execute delega al repo y mapea errores al transporte.
func (u ListStructures) Execute(ctx context.Context) (ListResponse, error) {
	items, err := u.Repo.ListActive(ctx)
	if err != nil {
		return ListResponse{}, apperrors.Internal("failed to list structures")
	}
	return ListResponse{Items: items, Total: len(items)}, nil
}

// GetStructure devuelve una estructura por id.
type GetStructure struct {
	Repo domain.StructureRepository
}

// Execute valida el id (no vacio) y delega al repo.
func (u GetStructure) Execute(ctx context.Context, id string) (entities.Structure, error) {
	if strings.TrimSpace(id) == "" {
		return entities.Structure{}, apperrors.BadRequest("id is required")
	}
	s, err := u.Repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrStructureNotFound) {
			return entities.Structure{}, apperrors.NotFound("structure not found")
		}
		return entities.Structure{}, apperrors.Internal("failed to load structure")
	}
	return s, nil
}

// CreateStructureInput es el input del usecase CreateStructure.
type CreateStructureInput struct {
	Name        string
	Type        string
	ParentID    *string
	Description string
	OrderIndex  int32
	ActorID     string
}

// CreateStructure crea una estructura nueva.
type CreateStructure struct {
	Repo domain.StructureRepository
}

// Execute valida y delega.
func (u CreateStructure) Execute(ctx context.Context, in CreateStructureInput) (entities.Structure, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return entities.Structure{}, apperrors.BadRequest("name is required")
	}
	if len(name) > nameMaxLen {
		return entities.Structure{}, apperrors.BadRequest("name too long")
	}
	t := entities.StructureType(in.Type)
	if !t.IsValid() {
		return entities.Structure{}, apperrors.BadRequest("invalid type")
	}
	if in.ParentID != nil && strings.TrimSpace(*in.ParentID) == "" {
		return entities.Structure{}, apperrors.BadRequest("parent_id must be a valid uuid or omitted")
	}
	s, err := u.Repo.Create(ctx, domain.CreateStructureInput{
		Name:        name,
		Type:        t,
		ParentID:    in.ParentID,
		Description: in.Description,
		OrderIndex:  in.OrderIndex,
		ActorID:     in.ActorID,
	})
	if err != nil {
		return entities.Structure{}, apperrors.Internal("failed to create structure")
	}
	return s, nil
}

// UpdateStructureInput es el input del usecase UpdateStructure.
type UpdateStructureInput struct {
	ID              string
	Name            string
	Type            string
	ParentID        *string
	Description     string
	OrderIndex      int32
	ActorID         string
	ExpectedVersion int32
}

// UpdateStructure actualiza una estructura existente.
type UpdateStructure struct {
	Repo domain.StructureRepository
}

// Execute valida y delega.
func (u UpdateStructure) Execute(ctx context.Context, in UpdateStructureInput) (entities.Structure, error) {
	if strings.TrimSpace(in.ID) == "" {
		return entities.Structure{}, apperrors.BadRequest("id is required")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return entities.Structure{}, apperrors.BadRequest("name is required")
	}
	if len(name) > nameMaxLen {
		return entities.Structure{}, apperrors.BadRequest("name too long")
	}
	t := entities.StructureType(in.Type)
	if !t.IsValid() {
		return entities.Structure{}, apperrors.BadRequest("invalid type")
	}
	if in.ExpectedVersion <= 0 {
		return entities.Structure{}, apperrors.BadRequest("expected_version must be > 0")
	}
	s, err := u.Repo.Update(ctx, domain.UpdateStructureInput{
		ID:              in.ID,
		Name:            name,
		Type:            t,
		ParentID:        in.ParentID,
		Description:     in.Description,
		OrderIndex:      in.OrderIndex,
		ActorID:         in.ActorID,
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrStructureNotFound):
			return entities.Structure{}, apperrors.NotFound("structure not found")
		case errors.Is(err, domain.ErrVersionMismatch):
			return entities.Structure{}, apperrors.Conflict("structure version mismatch")
		default:
			return entities.Structure{}, apperrors.Internal("failed to update structure")
		}
	}
	return s, nil
}

// ArchiveStructure hace soft-delete por id.
type ArchiveStructure struct {
	Repo domain.StructureRepository
}

// Execute valida y delega.
func (u ArchiveStructure) Execute(ctx context.Context, id, actorID string) error {
	if strings.TrimSpace(id) == "" {
		return apperrors.BadRequest("id is required")
	}
	if err := u.Repo.Archive(ctx, id, actorID); err != nil {
		if errors.Is(err, domain.ErrStructureNotFound) {
			return apperrors.NotFound("structure not found")
		}
		return apperrors.Internal("failed to archive structure")
	}
	return nil
}
