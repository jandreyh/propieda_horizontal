package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/units/application/dto"
	"github.com/saas-ph/api/internal/modules/units/domain"
	"github.com/saas-ph/api/internal/modules/units/domain/policies"
)

// AddOwnerToUnitDeps agrupa las dependencias del usecase AddOwnerToUnit.
type AddOwnerToUnitDeps struct {
	Units  domain.UnitRepository
	Owners domain.OwnerRepository
	Now    func() time.Time
}

// AddOwnerToUnitUseCase implementa POST /units/{id}/owners.
type AddOwnerToUnitUseCase struct {
	deps AddOwnerToUnitDeps
}

// NewAddOwnerToUnitUseCase construye el usecase.
func NewAddOwnerToUnitUseCase(deps AddOwnerToUnitDeps) *AddOwnerToUnitUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &AddOwnerToUnitUseCase{deps: deps}
}

// AddOwnerToUnitInput agrupa la entrada del usecase.
type AddOwnerToUnitInput struct {
	UnitID      string
	UserID      string
	Percentage  float64
	SinceDate   *string // YYYY-MM-DD opcional
	ActorUserID *string
}

// Execute valida invariantes (% acumulado <=100, unidad existe) y crea
// la fila en unit_owners.
func (uc *AddOwnerToUnitUseCase) Execute(ctx context.Context, in AddOwnerToUnitInput) (dto.OwnerDTO, error) {
	if strings.TrimSpace(in.UnitID) == "" || strings.TrimSpace(in.UserID) == "" {
		return dto.OwnerDTO{}, ErrInvalidInput
	}
	if !policies.ValidatePercentageRange(in.Percentage) {
		return dto.OwnerDTO{}, errorsJoin(ErrInvalidInput, domain.ErrInvalidPercentage)
	}

	// 1. La unidad debe existir (no soft-deleted).
	if _, err := uc.deps.Units.GetByID(ctx, in.UnitID); err != nil {
		if errors.Is(err, domain.ErrUnitNotFound) {
			return dto.OwnerDTO{}, errorsJoin(ErrNotFound, err)
		}
		return dto.OwnerDTO{}, errorsJoin(ErrInternal, err)
	}

	// 2. Cargar owners activos para validar suma de %.
	current, err := uc.deps.Owners.ListActive(ctx, in.UnitID)
	if err != nil {
		return dto.OwnerDTO{}, errorsJoin(ErrInternal, err)
	}
	if !policies.ValidatePercentage(current, in.Percentage) {
		return dto.OwnerDTO{}, errorsJoin(ErrPolicyRejected, domain.ErrInvalidPercentage)
	}

	// 3. Resolver fecha.
	since, err := parseDateOrFallback(in.SinceDate, uc.deps.Now().UTC())
	if err != nil {
		return dto.OwnerDTO{}, err
	}

	// 4. Crear.
	owner, err := uc.deps.Owners.Add(ctx, domain.AddOwnerParams{
		UnitID:     in.UnitID,
		UserID:     in.UserID,
		Percentage: in.Percentage,
		SinceDate:  since,
		CreatedBy:  in.ActorUserID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrOwnerDuplicateActive) {
			return dto.OwnerDTO{}, errorsJoin(ErrConflict, err)
		}
		return dto.OwnerDTO{}, errorsJoin(ErrInternal, err)
	}
	return ownerToDTO(owner), nil
}

// TerminateOwnershipDeps agrupa las dependencias del usecase
// TerminateOwnership.
type TerminateOwnershipDeps struct {
	Owners domain.OwnerRepository
	Now    func() time.Time
}

// TerminateOwnershipUseCase implementa DELETE /units/{id}/owners/{ownerID}.
// La eliminacion logica del propietario se modela seteando until_date=now,
// no un soft-delete. Esto preserva el historico de copropiedad.
type TerminateOwnershipUseCase struct {
	deps TerminateOwnershipDeps
}

// NewTerminateOwnershipUseCase construye el usecase.
func NewTerminateOwnershipUseCase(deps TerminateOwnershipDeps) *TerminateOwnershipUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &TerminateOwnershipUseCase{deps: deps}
}

// TerminateOwnershipInput agrupa la entrada del usecase.
type TerminateOwnershipInput struct {
	OwnerID     string
	ActorUserID *string
}

// Execute marca la fila de propiedad como cerrada en `now`.
func (uc *TerminateOwnershipUseCase) Execute(ctx context.Context, in TerminateOwnershipInput) error {
	if strings.TrimSpace(in.OwnerID) == "" {
		return ErrInvalidInput
	}
	if err := uc.deps.Owners.Terminate(ctx, in.OwnerID, uc.deps.Now().UTC(), in.ActorUserID); err != nil {
		if errors.Is(err, domain.ErrOwnerNotFound) {
			return errorsJoin(ErrNotFound, err)
		}
		return errorsJoin(ErrInternal, err)
	}
	return nil
}
