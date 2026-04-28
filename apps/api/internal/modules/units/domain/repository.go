// Package domain expone las interfaces y errores de dominio del modulo
// units (unidades, propietarios, ocupantes). La inversion de
// dependencias es estricta: nada en este paquete importa
// infraestructura.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/units/domain/entities"
)

// Errores de dominio del modulo units. Los handlers HTTP los mapean a
// Problem+JSON conservando el codigo de estado adecuado.
var (
	// ErrUnitNotFound se devuelve cuando no existe la unidad solicitada.
	ErrUnitNotFound = errors.New("units: unit not found")

	// ErrOwnerNotFound se devuelve cuando no existe la fila de propiedad.
	ErrOwnerNotFound = errors.New("units: owner not found")

	// ErrOccupancyNotFound se devuelve cuando no existe la ocupacion.
	ErrOccupancyNotFound = errors.New("units: occupancy not found")

	// ErrUnitCodeTaken se devuelve cuando el code esta repetido dentro
	// de la misma estructura (o en el set sin estructura).
	ErrUnitCodeTaken = errors.New("units: code already taken in structure")

	// ErrInvalidPercentage se devuelve cuando la suma de porcentajes de
	// owners activos excederia 100, o cuando el % de la fila esta
	// fuera de (0, 100].
	ErrInvalidPercentage = errors.New("units: invalid ownership percentage")

	// ErrInvalidUnitType se devuelve para tipos no soportados.
	ErrInvalidUnitType = errors.New("units: invalid unit type")

	// ErrInvalidOccupancyRole se devuelve para roles no soportados.
	ErrInvalidOccupancyRole = errors.New("units: invalid occupancy role")

	// ErrPrimaryOccupantConflict se devuelve cuando se intenta marcar un
	// segundo occupant primario activo en la misma unidad.
	ErrPrimaryOccupantConflict = errors.New("units: another active primary occupant exists")

	// ErrOwnerDuplicateActive se devuelve cuando se intenta agregar al
	// mismo user como propietario activo dos veces.
	ErrOwnerDuplicateActive = errors.New("units: user is already an active owner of this unit")
)

// CreateUnitParams agrupa los datos para crear una unidad.
type CreateUnitParams struct {
	StructureID *string
	Code        string
	Type        entities.UnitType
	AreaM2      *float64
	Bedrooms    *int
	Coefficient *float64
	CreatedBy   *string
}

// AddOwnerParams agrupa los datos para registrar un propietario.
type AddOwnerParams struct {
	UnitID     string
	UserID     string
	Percentage float64
	SinceDate  time.Time
	CreatedBy  *string
}

// AddOccupantParams agrupa los datos para registrar un ocupante.
type AddOccupantParams struct {
	UnitID     string
	UserID     string
	Role       entities.OccupancyRole
	IsPrimary  bool
	MoveInDate time.Time
	CreatedBy  *string
}

// UnitRepository abstrae la tabla units.
type UnitRepository interface {
	Create(ctx context.Context, p CreateUnitParams) (entities.Unit, error)
	GetByID(ctx context.Context, id string) (entities.Unit, error)
	ListByStructure(ctx context.Context, structureID string) ([]entities.Unit, error)
	ListAll(ctx context.Context) ([]entities.Unit, error)
}

// OwnerRepository abstrae la tabla unit_owners.
type OwnerRepository interface {
	Add(ctx context.Context, p AddOwnerParams) (entities.UnitOwner, error)
	ListActive(ctx context.Context, unitID string) ([]entities.UnitOwner, error)
	Terminate(ctx context.Context, ownerID string, until time.Time, by *string) error
}

// OccupancyRepository abstrae la tabla unit_occupancies.
type OccupancyRepository interface {
	Add(ctx context.Context, p AddOccupantParams) (entities.UnitOccupancy, error)
	ListActive(ctx context.Context, unitID string) ([]entities.UnitOccupancy, error)
	MoveOut(ctx context.Context, occupancyID string, at time.Time, by *string) error
}

// PeopleByUnitRepository devuelve la lista combinada de owners activos
// + occupants activos para una unidad. Esta es la query critica de la
// fase: un solo round-trip a la base de datos.
type PeopleByUnitRepository interface {
	GetActivePeople(ctx context.Context, unitID string) ([]entities.PersonInUnit, error)
}
