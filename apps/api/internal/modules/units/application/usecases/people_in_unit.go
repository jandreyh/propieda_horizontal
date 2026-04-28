package usecases

import (
	"context"
	"errors"
	"strings"

	"github.com/saas-ph/api/internal/modules/units/application/dto"
	"github.com/saas-ph/api/internal/modules/units/domain"
)

// GetPeopleInUnitDeps agrupa dependencias del usecase critico.
type GetPeopleInUnitDeps struct {
	People domain.PeopleByUnitRepository
}

// GetPeopleInUnitUseCase implementa GET /units/{id}/people. Es el caso
// de uso CRITICO de la fase: dado un unit_id devuelve la lista
// combinada de propietarios activos + ocupantes activos en una sola
// consulta a la base de datos.
type GetPeopleInUnitUseCase struct {
	deps GetPeopleInUnitDeps
}

// NewGetPeopleInUnitUseCase construye el usecase.
func NewGetPeopleInUnitUseCase(deps GetPeopleInUnitDeps) *GetPeopleInUnitUseCase {
	return &GetPeopleInUnitUseCase{deps: deps}
}

// Execute resuelve la lista. Si el unit_id no existe (la query no
// devuelve filas), se devuelve la lista vacia con HTTP 200 desde el
// handler — la inexistencia de unidad no se confunde con "unidad
// vacia", por lo que el handler hace un GET previo cuando lo necesita.
func (uc *GetPeopleInUnitUseCase) Execute(ctx context.Context, unitID string) (dto.PeopleInUnitResponse, error) {
	if strings.TrimSpace(unitID) == "" {
		return dto.PeopleInUnitResponse{}, ErrInvalidInput
	}
	people, err := uc.deps.People.GetActivePeople(ctx, unitID)
	if err != nil {
		if errors.Is(err, domain.ErrUnitNotFound) {
			return dto.PeopleInUnitResponse{}, errorsJoin(ErrNotFound, err)
		}
		return dto.PeopleInUnitResponse{}, errorsJoin(ErrInternal, err)
	}
	out := dto.PeopleInUnitResponse{
		UnitID: unitID,
		People: make([]dto.PersonInUnitDTO, 0, len(people)),
	}
	for i := range people {
		out.People = append(out.People, dto.PersonInUnitDTO{
			UserID:     people[i].UserID,
			FullName:   people[i].FullName,
			Document:   people[i].Document,
			RoleInUnit: string(people[i].RoleInUnit),
			IsPrimary:  people[i].IsPrimary,
			SinceDate:  people[i].SinceDate,
		})
	}
	return out, nil
}
