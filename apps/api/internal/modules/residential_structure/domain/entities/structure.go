// Package entities define las entidades de dominio del modulo
// residential_structure.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos; nunca aparece como
//     columna ni como campo de dominio.
package entities

import "time"

// StructureType enumera los tipos validos de una estructura residencial.
type StructureType string

const (
	// StructureTypeTower representa una torre.
	StructureTypeTower StructureType = "tower"
	// StructureTypeBlock representa un bloque.
	StructureTypeBlock StructureType = "block"
	// StructureTypeStage representa una etapa.
	StructureTypeStage StructureType = "stage"
	// StructureTypeSection representa una seccion.
	StructureTypeSection StructureType = "section"
	// StructureTypeOther representa cualquier otra agrupacion.
	StructureTypeOther StructureType = "other"
)

// IsValid indica si el tipo es uno de los enumerados validos.
func (t StructureType) IsValid() bool {
	switch t {
	case StructureTypeTower, StructureTypeBlock, StructureTypeStage,
		StructureTypeSection, StructureTypeOther:
		return true
	}
	return false
}

// StructureStatus enumera los estados validos de una Structure.
type StructureStatus string

const (
	// StructureStatusActive marca la estructura como vigente.
	StructureStatusActive StructureStatus = "active"
	// StructureStatusArchived marca la estructura como soft-deleted.
	StructureStatusArchived StructureStatus = "archived"
)

// Structure representa un nodo del arbol estructural del conjunto
// (torre, bloque, etapa, seccion, otro). El arbol es opcional: ParentID
// nil indica una raiz.
type Structure struct {
	ID          string
	Name        string
	Type        StructureType
	ParentID    *string
	Description string
	OrderIndex  int32
	Status      StructureStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}

// IsArchived indica si la Structure esta soft-deleted.
func (s Structure) IsArchived() bool {
	return s.Status == StructureStatusArchived || s.DeletedAt != nil
}
