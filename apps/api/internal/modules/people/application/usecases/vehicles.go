// Package usecases orquesta la logica de aplicacion del modulo people.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/people/domain"
	"github.com/saas-ph/api/internal/modules/people/domain/entities"
	"github.com/saas-ph/api/internal/modules/people/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// CreateVehicle crea un vehiculo nuevo.
//
// Reglas:
//   - Normaliza la placa con NormalizePlate (trim + uppercase).
//   - Valida formato Colombiano (ABC123 o ABC12A).
//   - Valida tipo y anio.
type CreateVehicle struct {
	Repo domain.VehicleRepository
}

// CreateVehicleInput es el input del usecase (sin tags JSON).
type CreateVehicleInput struct {
	Plate   string
	Type    string
	Brand   *string
	Model   *string
	Color   *string
	Year    *int32
	ActorID string
}

// Execute valida y delega al repo.
func (u CreateVehicle) Execute(ctx context.Context, in CreateVehicleInput) (entities.Vehicle, error) {
	plate, err := policies.ValidatePlate(in.Plate)
	if err != nil {
		return entities.Vehicle{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateVehicleType(in.Type); err != nil {
		return entities.Vehicle{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateVehicleYear(in.Year); err != nil {
		return entities.Vehicle{}, apperrors.BadRequest(err.Error())
	}

	v, err := u.Repo.Create(ctx, domain.CreateVehicleInput{
		Plate:   plate,
		Type:    entities.VehicleType(in.Type),
		Brand:   in.Brand,
		Model:   in.Model,
		Color:   in.Color,
		Year:    in.Year,
		ActorID: in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPlateAlreadyExists):
			return entities.Vehicle{}, apperrors.Conflict("plate already exists")
		default:
			return entities.Vehicle{}, apperrors.Internal("failed to create vehicle")
		}
	}
	return v, nil
}

// GetVehicle devuelve un vehiculo por id.
type GetVehicle struct {
	Repo domain.VehicleRepository
}

// Execute valida id y delega al repo.
func (u GetVehicle) Execute(ctx context.Context, id string) (entities.Vehicle, error) {
	if err := policies.ValidateUUID(id); err != nil {
		return entities.Vehicle{}, apperrors.BadRequest(err.Error())
	}
	v, err := u.Repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrVehicleNotFound) {
			return entities.Vehicle{}, apperrors.NotFound("vehicle not found")
		}
		return entities.Vehicle{}, apperrors.Internal("failed to load vehicle")
	}
	return v, nil
}

// GetVehicleByPlate devuelve un vehiculo por placa (normalizada).
type GetVehicleByPlate struct {
	Repo domain.VehicleRepository
}

// Execute normaliza, valida y delega al repo.
func (u GetVehicleByPlate) Execute(ctx context.Context, plate string) (entities.Vehicle, error) {
	p, err := policies.ValidatePlate(plate)
	if err != nil {
		return entities.Vehicle{}, apperrors.BadRequest(err.Error())
	}
	v, err := u.Repo.GetByPlate(ctx, p)
	if err != nil {
		if errors.Is(err, domain.ErrVehicleNotFound) {
			return entities.Vehicle{}, apperrors.NotFound("vehicle not found")
		}
		return entities.Vehicle{}, apperrors.Internal("failed to load vehicle")
	}
	return v, nil
}
