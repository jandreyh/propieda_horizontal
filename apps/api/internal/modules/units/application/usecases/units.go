// Package usecases agrupa la orquestacion de los casos de uso del
// modulo units. Cada usecase recibe sus dependencias por inyeccion y
// no conoce HTTP ni la implementacion concreta de los repositorios.
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

// Errores publicos de los usecases. Los handlers HTTP los traducen a
// Problem+JSON conservando el codigo de estado adecuado.
var (
	ErrInvalidInput   = errors.New("units: invalid input")
	ErrInternal       = errors.New("units: internal error")
	ErrNotFound       = errors.New("units: not found")
	ErrConflict       = errors.New("units: conflict")
	ErrPolicyRejected = errors.New("units: policy rejected")
)

// dateLayout es el layout aceptado en los DTOs para campos *_date.
const dateLayout = "2006-01-02"

// CreateUnitDeps agrupa las dependencias del usecase CreateUnit.
type CreateUnitDeps struct {
	Units domain.UnitRepository
	Now   func() time.Time
}

// CreateUnitUseCase implementa POST /units.
type CreateUnitUseCase struct {
	deps CreateUnitDeps
}

// NewCreateUnitUseCase construye el usecase. Si Now es nil, usa time.Now.
func NewCreateUnitUseCase(deps CreateUnitDeps) *CreateUnitUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &CreateUnitUseCase{deps: deps}
}

// CreateUnitInput es el contrato de entrada del usecase.
type CreateUnitInput struct {
	StructureID *string
	Code        string
	Type        string
	AreaM2      *float64
	Bedrooms    *int
	Coefficient *float64
	ActorUserID *string
}

// Execute valida y crea la unidad.
func (uc *CreateUnitUseCase) Execute(ctx context.Context, in CreateUnitInput) (dto.UnitDTO, error) {
	code := strings.TrimSpace(in.Code)
	if code == "" {
		return dto.UnitDTO{}, ErrInvalidInput
	}
	t := entities.UnitType(strings.TrimSpace(in.Type))
	if !t.IsValid() {
		return dto.UnitDTO{}, errorsJoin(ErrInvalidInput, domain.ErrInvalidUnitType)
	}
	if in.Bedrooms != nil && *in.Bedrooms < 0 {
		return dto.UnitDTO{}, ErrInvalidInput
	}
	if in.AreaM2 != nil && *in.AreaM2 < 0 {
		return dto.UnitDTO{}, ErrInvalidInput
	}
	if in.Coefficient != nil && (*in.Coefficient < 0 || *in.Coefficient > 1) {
		return dto.UnitDTO{}, ErrInvalidInput
	}

	u, err := uc.deps.Units.Create(ctx, domain.CreateUnitParams{
		StructureID: in.StructureID,
		Code:        code,
		Type:        t,
		AreaM2:      in.AreaM2,
		Bedrooms:    in.Bedrooms,
		Coefficient: in.Coefficient,
		CreatedBy:   in.ActorUserID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnitCodeTaken) {
			return dto.UnitDTO{}, errorsJoin(ErrConflict, err)
		}
		return dto.UnitDTO{}, errorsJoin(ErrInternal, err)
	}
	return unitToDTO(u), nil
}

// GetUnitDeps agrupa las dependencias del usecase GetUnit.
type GetUnitDeps struct {
	Units domain.UnitRepository
}

// GetUnitUseCase implementa GET /units/{id}.
type GetUnitUseCase struct {
	deps GetUnitDeps
}

// NewGetUnitUseCase construye el usecase.
func NewGetUnitUseCase(deps GetUnitDeps) *GetUnitUseCase {
	return &GetUnitUseCase{deps: deps}
}

// Execute resuelve la unidad por id.
func (uc *GetUnitUseCase) Execute(ctx context.Context, id string) (dto.UnitDTO, error) {
	if strings.TrimSpace(id) == "" {
		return dto.UnitDTO{}, ErrInvalidInput
	}
	u, err := uc.deps.Units.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUnitNotFound) {
			return dto.UnitDTO{}, errorsJoin(ErrNotFound, err)
		}
		return dto.UnitDTO{}, errorsJoin(ErrInternal, err)
	}
	return unitToDTO(u), nil
}

// ListUnitsDeps agrupa las dependencias del usecase ListUnits.
type ListUnitsDeps struct {
	Units domain.UnitRepository
}

// ListUnitsUseCase implementa GET /units (con filtro opcional por
// structure_id).
type ListUnitsUseCase struct {
	deps ListUnitsDeps
}

// NewListUnitsUseCase construye el usecase.
func NewListUnitsUseCase(deps ListUnitsDeps) *ListUnitsUseCase {
	return &ListUnitsUseCase{deps: deps}
}

// ListUnitsInput es el filtro opcional aceptado por el usecase.
type ListUnitsInput struct {
	StructureID *string
}

// Execute lista unidades activas. Si StructureID es no-nil filtra por
// torre; en caso contrario devuelve todas las unidades activas del
// tenant.
func (uc *ListUnitsUseCase) Execute(ctx context.Context, in ListUnitsInput) (dto.ListUnitsResponse, error) {
	var (
		list []entities.Unit
		err  error
	)
	if in.StructureID != nil && *in.StructureID != "" {
		list, err = uc.deps.Units.ListByStructure(ctx, *in.StructureID)
	} else {
		list, err = uc.deps.Units.ListAll(ctx)
	}
	if err != nil {
		return dto.ListUnitsResponse{}, errorsJoin(ErrInternal, err)
	}
	out := dto.ListUnitsResponse{Items: make([]dto.UnitDTO, 0, len(list))}
	for i := range list {
		out.Items = append(out.Items, unitToDTO(list[i]))
	}
	return out, nil
}

// --- helpers compartidos ---

func unitToDTO(u entities.Unit) dto.UnitDTO {
	return dto.UnitDTO{
		ID:          u.ID,
		StructureID: u.StructureID,
		Code:        u.Code,
		Type:        string(u.Type),
		AreaM2:      u.AreaM2,
		Bedrooms:    u.Bedrooms,
		Coefficient: u.Coefficient,
		Status:      string(u.Status),
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		Version:     u.Version,
	}
}

func ownerToDTO(o entities.UnitOwner) dto.OwnerDTO {
	return dto.OwnerDTO{
		ID:         o.ID,
		UnitID:     o.UnitID,
		UserID:     o.UserID,
		Percentage: o.Percentage,
		SinceDate:  o.SinceDate,
		UntilDate:  o.UntilDate,
		Status:     string(o.Status),
		CreatedAt:  o.CreatedAt,
		UpdatedAt:  o.UpdatedAt,
		Version:    o.Version,
	}
}

func occupantToDTO(o entities.UnitOccupancy) dto.OccupantDTO {
	return dto.OccupantDTO{
		ID:          o.ID,
		UnitID:      o.UnitID,
		UserID:      o.UserID,
		RoleInUnit:  string(o.RoleInUnit),
		IsPrimary:   o.IsPrimary,
		MoveInDate:  o.MoveInDate,
		MoveOutDate: o.MoveOutDate,
		Status:      string(o.Status),
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
		Version:     o.Version,
	}
}

// errorsJoin envuelve outer y inner para preservar la cadena al
// hacer errors.Is desde el handler.
func errorsJoin(outer, inner error) error {
	return wrap{outer: outer, inner: inner}
}

type wrap struct {
	outer, inner error
}

func (w wrap) Error() string {
	if w.inner == nil {
		return w.outer.Error()
	}
	return w.outer.Error() + ": " + w.inner.Error()
}

func (w wrap) Unwrap() []error {
	return []error{w.outer, w.inner}
}

// parseDateOrNow interpreta un string YYYY-MM-DD; si esta vacio o nil,
// devuelve fallback.
func parseDateOrFallback(s *string, fallback time.Time) (time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return fallback, nil
	}
	t, err := time.Parse(dateLayout, strings.TrimSpace(*s))
	if err != nil {
		return time.Time{}, ErrInvalidInput
	}
	return t, nil
}

// _ ensures policies is referenced (some flow uses it directly).
var _ = policies.MaxOwnershipPercentage
