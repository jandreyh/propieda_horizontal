package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/units/application/dto"
	"github.com/saas-ph/api/internal/modules/units/domain"
	"github.com/saas-ph/api/internal/modules/units/domain/entities"
	"github.com/saas-ph/api/internal/modules/units/domain/policies"
)

// AddOccupantToUnitDeps agrupa dependencias del usecase
// AddOccupantToUnit.
type AddOccupantToUnitDeps struct {
	Units     domain.UnitRepository
	Occupants domain.OccupancyRepository
	Now       func() time.Time
}

// AddOccupantToUnitUseCase implementa POST /units/{id}/occupants.
type AddOccupantToUnitUseCase struct {
	deps AddOccupantToUnitDeps
}

// NewAddOccupantToUnitUseCase construye el usecase.
func NewAddOccupantToUnitUseCase(deps AddOccupantToUnitDeps) *AddOccupantToUnitUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &AddOccupantToUnitUseCase{deps: deps}
}

// AddOccupantToUnitInput agrupa la entrada del usecase.
type AddOccupantToUnitInput struct {
	UnitID      string
	UserID      string
	Role        string
	IsPrimary   bool
	MoveInDate  *string // YYYY-MM-DD opcional
	ActorUserID *string
}

// Execute valida invariantes (rol valido, unidad existe, primary unico)
// y crea la fila en unit_occupancies.
func (uc *AddOccupantToUnitUseCase) Execute(ctx context.Context, in AddOccupantToUnitInput) (dto.OccupantDTO, error) {
	if strings.TrimSpace(in.UnitID) == "" || strings.TrimSpace(in.UserID) == "" {
		return dto.OccupantDTO{}, ErrInvalidInput
	}
	role := entities.OccupancyRole(strings.TrimSpace(in.Role))
	if !role.IsValid() {
		return dto.OccupantDTO{}, errorsJoin(ErrInvalidInput, domain.ErrInvalidOccupancyRole)
	}

	// 1. La unidad debe existir.
	if _, err := uc.deps.Units.GetByID(ctx, in.UnitID); err != nil {
		if errors.Is(err, domain.ErrUnitNotFound) {
			return dto.OccupantDTO{}, errorsJoin(ErrNotFound, err)
		}
		return dto.OccupantDTO{}, errorsJoin(ErrInternal, err)
	}

	// 2. Si IsPrimary=true, no debe haber otro primary activo.
	if in.IsPrimary {
		current, err := uc.deps.Occupants.ListActive(ctx, in.UnitID)
		if err != nil {
			return dto.OccupantDTO{}, errorsJoin(ErrInternal, err)
		}
		if !policies.EnsureOnlyOnePrimary(current, true) {
			return dto.OccupantDTO{}, errorsJoin(ErrPolicyRejected, domain.ErrPrimaryOccupantConflict)
		}
	}

	// 3. Resolver fecha.
	moveIn, err := parseDateOrFallback(in.MoveInDate, uc.deps.Now().UTC())
	if err != nil {
		return dto.OccupantDTO{}, err
	}

	// 4. Crear.
	occ, err := uc.deps.Occupants.Add(ctx, domain.AddOccupantParams{
		UnitID:     in.UnitID,
		UserID:     in.UserID,
		Role:       role,
		IsPrimary:  in.IsPrimary,
		MoveInDate: moveIn,
		CreatedBy:  in.ActorUserID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrPrimaryOccupantConflict) {
			return dto.OccupantDTO{}, errorsJoin(ErrConflict, err)
		}
		return dto.OccupantDTO{}, errorsJoin(ErrInternal, err)
	}
	return occupantToDTO(occ), nil
}

// MoveOutOccupantDeps agrupa dependencias del usecase MoveOutOccupant.
type MoveOutOccupantDeps struct {
	Occupants domain.OccupancyRepository
	Now       func() time.Time
}

// MoveOutOccupantUseCase implementa DELETE
// /units/{id}/occupants/{occupancyID}. Marca move_out_date=now (no
// soft-delete) para preservar el historico.
type MoveOutOccupantUseCase struct {
	deps MoveOutOccupantDeps
}

// NewMoveOutOccupantUseCase construye el usecase.
func NewMoveOutOccupantUseCase(deps MoveOutOccupantDeps) *MoveOutOccupantUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &MoveOutOccupantUseCase{deps: deps}
}

// MoveOutOccupantInput agrupa la entrada del usecase.
type MoveOutOccupantInput struct {
	OccupancyID string
	ActorUserID *string
}

// Execute marca la ocupacion como cerrada en `now`.
func (uc *MoveOutOccupantUseCase) Execute(ctx context.Context, in MoveOutOccupantInput) error {
	if strings.TrimSpace(in.OccupancyID) == "" {
		return ErrInvalidInput
	}
	if err := uc.deps.Occupants.MoveOut(ctx, in.OccupancyID, uc.deps.Now().UTC(), in.ActorUserID); err != nil {
		if errors.Is(err, domain.ErrOccupancyNotFound) {
			return errorsJoin(ErrNotFound, err)
		}
		return errorsJoin(ErrInternal, err)
	}
	return nil
}
