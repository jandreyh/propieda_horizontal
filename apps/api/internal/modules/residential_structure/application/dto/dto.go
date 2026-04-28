// Package dto contiene los Data Transfer Objects del modulo
// residential_structure. Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// StructureResponse es la representacion HTTP de una entities.Structure.
type StructureResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Description string    `json:"description,omitempty"`
	OrderIndex  int32     `json:"order_index"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int32     `json:"version"`
}

// ListStructuresResponse es el sobre del listado.
type ListStructuresResponse struct {
	Items []StructureResponse `json:"items"`
	Total int                 `json:"total"`
}

// CreateStructureRequest es el body de POST /structures.
type CreateStructureRequest struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	ParentID    *string `json:"parent_id,omitempty"`
	Description string  `json:"description,omitempty"`
	OrderIndex  int32   `json:"order_index"`
}

// UpdateStructureRequest es el body de PUT /structures/{id}.
type UpdateStructureRequest struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	ParentID        *string `json:"parent_id,omitempty"`
	Description     string  `json:"description,omitempty"`
	OrderIndex      int32   `json:"order_index"`
	ExpectedVersion int32   `json:"expected_version"`
}
