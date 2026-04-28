// Package dto agrupa los Data Transfer Objects HTTP del modulo units.
//
// Las entidades del dominio NO llevan tags JSON. Aqui SI viven los tags
// para serializar y deserializar requests/responses HTTP.
package dto

import "time"

// CreateUnitRequest es el body de POST /units.
type CreateUnitRequest struct {
	StructureID *string  `json:"structure_id,omitempty"`
	Code        string   `json:"code"`
	Type        string   `json:"type"`
	AreaM2      *float64 `json:"area_m2,omitempty"`
	Bedrooms    *int     `json:"bedrooms,omitempty"`
	Coefficient *float64 `json:"coefficient,omitempty"`
}

// UnitDTO es la representacion HTTP de una unidad.
type UnitDTO struct {
	ID          string    `json:"id"`
	StructureID *string   `json:"structure_id,omitempty"`
	Code        string    `json:"code"`
	Type        string    `json:"type"`
	AreaM2      *float64  `json:"area_m2,omitempty"`
	Bedrooms    *int      `json:"bedrooms,omitempty"`
	Coefficient *float64  `json:"coefficient,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
}

// AddOwnerRequest es el body de POST /units/{id}/owners.
type AddOwnerRequest struct {
	UserID     string  `json:"user_id"`
	Percentage float64 `json:"percentage"`
	SinceDate  *string `json:"since_date,omitempty"` // YYYY-MM-DD opcional
}

// OwnerDTO es la representacion HTTP de una propiedad.
type OwnerDTO struct {
	ID         string     `json:"id"`
	UnitID     string     `json:"unit_id"`
	UserID     string     `json:"user_id"`
	Percentage float64    `json:"percentage"`
	SinceDate  time.Time  `json:"since_date"`
	UntilDate  *time.Time `json:"until_date,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Version    int        `json:"version"`
}

// AddOccupantRequest es el body de POST /units/{id}/occupants.
type AddOccupantRequest struct {
	UserID     string  `json:"user_id"`
	Role       string  `json:"role_in_unit"`
	IsPrimary  bool    `json:"is_primary"`
	MoveInDate *string `json:"move_in_date,omitempty"` // YYYY-MM-DD opcional
}

// OccupantDTO es la representacion HTTP de una ocupacion.
type OccupantDTO struct {
	ID          string     `json:"id"`
	UnitID      string     `json:"unit_id"`
	UserID      string     `json:"user_id"`
	RoleInUnit  string     `json:"role_in_unit"`
	IsPrimary   bool       `json:"is_primary"`
	MoveInDate  time.Time  `json:"move_in_date"`
	MoveOutDate *time.Time `json:"move_out_date,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Version     int        `json:"version"`
}

// PersonInUnitDTO es la representacion HTTP de una persona vinculada a
// la unidad (owner o occupant) para GET /units/{id}/people.
type PersonInUnitDTO struct {
	UserID     string    `json:"user_id"`
	FullName   string    `json:"full_name"`
	Document   string    `json:"document"`
	RoleInUnit string    `json:"role_in_unit"`
	IsPrimary  bool      `json:"is_primary"`
	SinceDate  time.Time `json:"since_date"`
}

// PeopleInUnitResponse envuelve la lista para GET /units/{id}/people.
type PeopleInUnitResponse struct {
	UnitID string            `json:"unit_id"`
	People []PersonInUnitDTO `json:"people"`
}

// ListUnitsResponse envuelve la lista para GET /units.
type ListUnitsResponse struct {
	Items []UnitDTO `json:"items"`
}
