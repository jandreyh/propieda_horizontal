// Package domain define las interfaces (puertos) del modulo
// residential_structure que la capa de aplicacion consume y que la
// infraestructura implementa.
package domain

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/residential_structure/domain/entities"
)

// ErrStructureNotFound se devuelve cuando un id consultado no existe (o
// esta archivado).
var ErrStructureNotFound = errors.New("residential_structure: structure not found")

// ErrVersionMismatch se devuelve cuando un update optimista no encuentra
// la version esperada (otro proceso actualizo entre la lectura y este
// write).
var ErrVersionMismatch = errors.New("residential_structure: version mismatch (concurrent update)")

// CreateStructureInput agrupa los datos requeridos para crear una
// estructura nueva.
type CreateStructureInput struct {
	Name        string
	Type        entities.StructureType
	ParentID    *string
	Description string
	OrderIndex  int32
	ActorID     string
}

// UpdateStructureInput agrupa los datos requeridos para actualizar una
// estructura existente. Concurrencia optimista por ExpectedVersion.
type UpdateStructureInput struct {
	ID              string
	Name            string
	Type            entities.StructureType
	ParentID        *string
	Description     string
	OrderIndex      int32
	ActorID         string
	ExpectedVersion int32
}

// StructureRepository es el puerto que persiste residential_structures.
type StructureRepository interface {
	// ListActive devuelve todas las estructuras activas ordenadas por
	// (order_index, name).
	ListActive(ctx context.Context) ([]entities.Structure, error)
	// GetByID devuelve la estructura activa con ese id, o
	// ErrStructureNotFound.
	GetByID(ctx context.Context, id string) (entities.Structure, error)
	// Create persiste una estructura nueva.
	Create(ctx context.Context, in CreateStructureInput) (entities.Structure, error)
	// Update actualiza una estructura existente. Si la fila no existe
	// devuelve ErrStructureNotFound; si la version no calza,
	// ErrVersionMismatch.
	Update(ctx context.Context, in UpdateStructureInput) (entities.Structure, error)
	// Archive marca la estructura como archived (soft-delete). Devuelve
	// ErrStructureNotFound si no existe o ya estaba archivada.
	Archive(ctx context.Context, id string, actorID string) error
	// ListChildren devuelve los hijos directos de la estructura padre
	// indicada (excluye archivadas), ordenados por (order_index, name).
	ListChildren(ctx context.Context, parentID string) ([]entities.Structure, error)
}
