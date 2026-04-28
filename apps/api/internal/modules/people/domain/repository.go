// Package domain define las interfaces (puertos) del modulo people que la
// capa de aplicacion consume y que infra implementa.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/people/domain/entities"
)

// ErrVehicleNotFound se devuelve cuando no existe (o esta archivado) un
// vehiculo por el id/placa consultados.
var ErrVehicleNotFound = errors.New("people: vehicle not found")

// ErrPlateAlreadyExists se devuelve cuando un Create choca con la
// restriccion UNIQUE de plate (vehiculo activo con misma placa).
var ErrPlateAlreadyExists = errors.New("people: plate already exists")

// ErrAssignmentNotFound se devuelve cuando una asignacion por id no
// existe o ya fue cerrada / archivada.
var ErrAssignmentNotFound = errors.New("people: assignment not found")

// ErrVehicleAlreadyAssigned se devuelve cuando se intenta crear una
// asignacion activa para un vehiculo que ya tiene otra activa.
var ErrVehicleAlreadyAssigned = errors.New("people: vehicle already assigned to a unit")

// CreateVehicleInput agrupa los datos necesarios para persistir un
// vehiculo nuevo. La placa debe llegar YA normalizada.
type CreateVehicleInput struct {
	Plate   string
	Type    entities.VehicleType
	Brand   *string
	Model   *string
	Color   *string
	Year    *int32
	ActorID string
}

// VehicleRepository es el puerto que persiste vehiculos.
type VehicleRepository interface {
	// Create inserta un vehiculo. Si la placa ya existe activamente,
	// devuelve ErrPlateAlreadyExists.
	Create(ctx context.Context, in CreateVehicleInput) (entities.Vehicle, error)
	// GetByID devuelve un vehiculo activo por id, o ErrVehicleNotFound.
	GetByID(ctx context.Context, id string) (entities.Vehicle, error)
	// GetByPlate devuelve un vehiculo activo por placa (ya normalizada),
	// o ErrVehicleNotFound.
	GetByPlate(ctx context.Context, plate string) (entities.Vehicle, error)
	// ListAll devuelve todos los vehiculos activos ordenados por placa.
	ListAll(ctx context.Context) ([]entities.Vehicle, error)
}

// AssignInput agrupa los datos necesarios para crear una asignacion.
type AssignInput struct {
	UnitID    string
	VehicleID string
	// SinceDate opcional; si nil el repo aplica CURRENT_DATE.
	SinceDate *time.Time
	ActorID   string
}

// EndAssignmentInput agrupa los datos para cerrar una asignacion.
type EndAssignmentInput struct {
	AssignmentID string
	// UntilDate opcional; si nil el repo aplica CURRENT_DATE.
	UntilDate *time.Time
	ActorID   string
}

// AssignmentRepository es el puerto que persiste asignaciones
// vehiculo<->unidad.
type AssignmentRepository interface {
	// Assign crea una asignacion activa. Devuelve ErrVehicleAlreadyAssigned
	// si el vehiculo ya tiene otra asignacion activa.
	Assign(ctx context.Context, in AssignInput) (entities.UnitVehicleAssignment, error)
	// ListActiveByUnit devuelve las asignaciones activas (sin until_date)
	// de la unidad, con el vehiculo materializado en cada item.
	ListActiveByUnit(ctx context.Context, unitID string) ([]entities.UnitVehicleAssignment, error)
	// End cierra una asignacion (until_date = today o el provisto).
	// Devuelve ErrAssignmentNotFound si no existe o ya fue cerrada.
	End(ctx context.Context, in EndAssignmentInput) (entities.UnitVehicleAssignment, error)
}
