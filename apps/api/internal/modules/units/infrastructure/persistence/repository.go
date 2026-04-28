// Package persistence implementa los repositorios del modulo units.
//
// Diseno: stateless. Cada metodo del repositorio resuelve el pool del
// tenant a partir del contexto del request via `tenantctx.FromCtx`.
// Esto encaja con el modelo multi-tenant DB-por-tenant: una unica
// instancia del repo sirve a todos los tenants y la conexion correcta
// se elige por request.
//
// IMPORTANTE: la implementacion concreta sobre PostgreSQL se materializa
// cuando sqlc genera el paquete `unitsdb` a partir de las queries en
// `queries/units.sql` y la migracion 005. Mientras tanto, los repos
// existen como stubs que respetan las interfaces del dominio para que
// los tests del DoD (policies + usecases con mocks) compilen y corran.
// El cableado en main.go puede inyectar estos stubs y obtener errores
// claros en runtime hasta completar la generacion de sqlc.
package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/units/domain"
	"github.com/saas-ph/api/internal/modules/units/domain/entities"
)

// errStub se devuelve mientras la implementacion concreta sobre sqlc no
// este disponible. Es el mismo patron usado en authorization durante el
// MVP: el modulo es 100% testeable end-to-end con mocks y la persistencia
// concreta se completa cuando sqlc.yaml haya generado el paquete unitsdb.
var errStub = errors.New("units persistence: implementacion concreta pendiente (sqlc gen)")

// UnitRepo implementa domain.UnitRepository.
type UnitRepo struct{}

// NewUnitRepo construye un UnitRepo sin estado.
func NewUnitRepo() *UnitRepo { return &UnitRepo{} }

// Create stub.
func (r *UnitRepo) Create(ctx context.Context, p domain.CreateUnitParams) (entities.Unit, error) {
	_ = ctx
	_ = p
	return entities.Unit{}, errStub
}

// GetByID stub.
func (r *UnitRepo) GetByID(ctx context.Context, id string) (entities.Unit, error) {
	_ = ctx
	_ = id
	return entities.Unit{}, errStub
}

// ListByStructure stub.
func (r *UnitRepo) ListByStructure(ctx context.Context, structureID string) ([]entities.Unit, error) {
	_ = ctx
	_ = structureID
	return nil, errStub
}

// ListAll stub.
func (r *UnitRepo) ListAll(ctx context.Context) ([]entities.Unit, error) {
	_ = ctx
	return nil, errStub
}

// OwnerRepo implementa domain.OwnerRepository.
type OwnerRepo struct{}

// NewOwnerRepo construye un OwnerRepo sin estado.
func NewOwnerRepo() *OwnerRepo { return &OwnerRepo{} }

// Add stub.
func (r *OwnerRepo) Add(ctx context.Context, p domain.AddOwnerParams) (entities.UnitOwner, error) {
	_ = ctx
	_ = p
	return entities.UnitOwner{}, errStub
}

// ListActive stub.
func (r *OwnerRepo) ListActive(ctx context.Context, unitID string) ([]entities.UnitOwner, error) {
	_ = ctx
	_ = unitID
	return nil, errStub
}

// Terminate stub.
func (r *OwnerRepo) Terminate(ctx context.Context, ownerID string, until time.Time, by *string) error {
	_ = ctx
	_ = ownerID
	_ = until
	_ = by
	return errStub
}

// OccupancyRepo implementa domain.OccupancyRepository.
type OccupancyRepo struct{}

// NewOccupancyRepo construye un OccupancyRepo sin estado.
func NewOccupancyRepo() *OccupancyRepo { return &OccupancyRepo{} }

// Add stub.
func (r *OccupancyRepo) Add(ctx context.Context, p domain.AddOccupantParams) (entities.UnitOccupancy, error) {
	_ = ctx
	_ = p
	return entities.UnitOccupancy{}, errStub
}

// ListActive stub.
func (r *OccupancyRepo) ListActive(ctx context.Context, unitID string) ([]entities.UnitOccupancy, error) {
	_ = ctx
	_ = unitID
	return nil, errStub
}

// MoveOut stub.
func (r *OccupancyRepo) MoveOut(ctx context.Context, occupancyID string, at time.Time, by *string) error {
	_ = ctx
	_ = occupancyID
	_ = at
	_ = by
	return errStub
}

// PeopleByUnitRepo implementa domain.PeopleByUnitRepository. La
// implementacion concreta debe invocar la query SQL `GetActivePeopleForUnit`
// del paquete `unitsdb` (sqlc), que devuelve owners activos + occupants
// activos en un solo round-trip. Mientras la generacion no este
// disponible, los tests del usecase critico usan un mock directo de esta
// interfaz (ver usecases/people_in_unit_test.go).
type PeopleByUnitRepo struct{}

// NewPeopleByUnitRepo construye un PeopleByUnitRepo sin estado.
func NewPeopleByUnitRepo() *PeopleByUnitRepo { return &PeopleByUnitRepo{} }

// GetActivePeople stub.
func (r *PeopleByUnitRepo) GetActivePeople(ctx context.Context, unitID string) ([]entities.PersonInUnit, error) {
	_ = ctx
	_ = unitID
	return nil, errStub
}

// Asegura en compile time que los repos implementan las interfaces del
// dominio. Si la firma divergiera, el build del modulo rompe.
var (
	_ domain.UnitRepository         = (*UnitRepo)(nil)
	_ domain.OwnerRepository        = (*OwnerRepo)(nil)
	_ domain.OccupancyRepository    = (*OccupancyRepo)(nil)
	_ domain.PeopleByUnitRepository = (*PeopleByUnitRepo)(nil)
)
