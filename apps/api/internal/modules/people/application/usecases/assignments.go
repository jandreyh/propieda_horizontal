package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/people/domain"
	"github.com/saas-ph/api/internal/modules/people/domain/entities"
	"github.com/saas-ph/api/internal/modules/people/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// AssignVehicleToUnit crea una asignacion activa vehiculo<->unidad.
type AssignVehicleToUnit struct {
	Repo domain.AssignmentRepository
}

// AssignVehicleInput es el input del usecase.
type AssignVehicleInput struct {
	UnitID    string
	VehicleID string
	SinceDate *time.Time
	ActorID   string
}

// Execute valida y delega al repo.
func (u AssignVehicleToUnit) Execute(ctx context.Context, in AssignVehicleInput) (entities.UnitVehicleAssignment, error) {
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.UnitVehicleAssignment{}, apperrors.BadRequest("unit_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.VehicleID); err != nil {
		return entities.UnitVehicleAssignment{}, apperrors.BadRequest("vehicle_id: " + err.Error())
	}
	a, err := u.Repo.Assign(ctx, domain.AssignInput{
		UnitID:    in.UnitID,
		VehicleID: in.VehicleID,
		SinceDate: in.SinceDate,
		ActorID:   in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrVehicleAlreadyAssigned):
			return entities.UnitVehicleAssignment{}, apperrors.Conflict("vehicle already assigned to a unit")
		case errors.Is(err, domain.ErrVehicleNotFound):
			return entities.UnitVehicleAssignment{}, apperrors.NotFound("vehicle not found")
		default:
			return entities.UnitVehicleAssignment{}, apperrors.Internal("failed to assign vehicle")
		}
	}
	return a, nil
}

// ListActiveVehiclesForUnit lista las asignaciones activas (con
// vehiculo materializado) de una unidad.
type ListActiveVehiclesForUnit struct {
	Repo domain.AssignmentRepository
}

// Execute valida y delega.
func (u ListActiveVehiclesForUnit) Execute(ctx context.Context, unitID string) ([]entities.UnitVehicleAssignment, error) {
	if err := policies.ValidateUUID(unitID); err != nil {
		return nil, apperrors.BadRequest("unit_id: " + err.Error())
	}
	out, err := u.Repo.ListActiveByUnit(ctx, unitID)
	if err != nil {
		return nil, apperrors.Internal("failed to list assignments")
	}
	return out, nil
}

// EndAssignment cierra una asignacion existente.
type EndAssignment struct {
	Repo domain.AssignmentRepository
	// Now permite inyectar reloj para tests; si nil, time.Now.
	Now func() time.Time
}

// EndAssignmentInput es el input del usecase.
type EndAssignmentInput struct {
	AssignmentID string
	UntilDate    *time.Time
	ActorID      string
}

// Execute valida y delega.
func (u EndAssignment) Execute(ctx context.Context, in EndAssignmentInput) (entities.UnitVehicleAssignment, error) {
	if err := policies.ValidateUUID(in.AssignmentID); err != nil {
		return entities.UnitVehicleAssignment{}, apperrors.BadRequest("assignment_id: " + err.Error())
	}
	a, err := u.Repo.End(ctx, domain.EndAssignmentInput{
		AssignmentID: in.AssignmentID,
		UntilDate:    in.UntilDate,
		ActorID:      in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrAssignmentNotFound) {
			return entities.UnitVehicleAssignment{}, apperrors.NotFound("assignment not found or already closed")
		}
		return entities.UnitVehicleAssignment{}, apperrors.Internal("failed to end assignment")
	}
	return a, nil
}
