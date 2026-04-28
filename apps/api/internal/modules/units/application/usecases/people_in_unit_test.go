package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/units/application/usecases"
	"github.com/saas-ph/api/internal/modules/units/domain"
	"github.com/saas-ph/api/internal/modules/units/domain/entities"
)

// peopleRepoMock implementa domain.PeopleByUnitRepository en memoria.
type peopleRepoMock struct {
	byUnit map[string][]entities.PersonInUnit
	err    error
	calls  int
}

func newPeopleRepoMock() *peopleRepoMock {
	return &peopleRepoMock{byUnit: map[string][]entities.PersonInUnit{}}
}

func (m *peopleRepoMock) GetActivePeople(_ context.Context, unitID string) ([]entities.PersonInUnit, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	out, ok := m.byUnit[unitID]
	if !ok {
		return []entities.PersonInUnit{}, nil
	}
	return out, nil
}

// TestGetPeopleInUnit_GoldenPath cubre: 2 owners activos + 1 occupant
// activo aparecen; el occupant con move_out_date NO debe llegar al
// usecase porque la query de DB ya lo filtra (el mock simula esa
// realidad omitiendo la fila terminada). Verifica que el handler de
// caso de uso devuelve los 3 esperados, en el orden recibido.
func TestGetPeopleInUnit_GoldenPath(t *testing.T) {
	mock := newPeopleRepoMock()
	since := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	mock.byUnit["unit-101"] = []entities.PersonInUnit{
		{
			UserID:     "user-A",
			FullName:   "Ana Perez",
			Document:   "CC:111",
			RoleInUnit: entities.PersonRoleOwner,
			IsPrimary:  false,
			SinceDate:  since,
		},
		{
			UserID:     "user-B",
			FullName:   "Beto Gomez",
			Document:   "CC:222",
			RoleInUnit: entities.PersonRoleOwner,
			IsPrimary:  false,
			SinceDate:  since,
		},
		{
			UserID:     "user-C",
			FullName:   "Carla Diaz",
			Document:   "CC:333",
			RoleInUnit: "tenant",
			IsPrimary:  true,
			SinceDate:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		// IMPORTANTE: NO incluimos a "user-D" con move_out_date porque
		// la query SQL ya los excluye via WHERE move_out_date IS NULL.
		// Si llegara a aparecer en la respuesta, el test fallaria.
	}

	uc := usecases.NewGetPeopleInUnitUseCase(usecases.GetPeopleInUnitDeps{People: mock})

	resp, err := uc.Execute(context.Background(), "unit-101")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.UnitID != "unit-101" {
		t.Fatalf("unit_id mismatch: %q", resp.UnitID)
	}
	if len(resp.People) != 3 {
		t.Fatalf("expected 3 people, got %d", len(resp.People))
	}

	// 2 owners
	owners := 0
	tenants := 0
	primaries := 0
	for _, p := range resp.People {
		if p.RoleInUnit == "owner" {
			owners++
		}
		if p.RoleInUnit == "tenant" {
			tenants++
		}
		if p.IsPrimary {
			primaries++
		}
	}
	if owners != 2 {
		t.Fatalf("expected 2 owners, got %d", owners)
	}
	if tenants != 1 {
		t.Fatalf("expected 1 tenant, got %d", tenants)
	}
	if primaries != 1 {
		t.Fatalf("expected exactly 1 primary, got %d", primaries)
	}

	// Detalle de ningun usuario "user-D" (el que tendria move_out_date).
	for _, p := range resp.People {
		if p.UserID == "user-D" {
			t.Fatalf("a moved-out occupant must not appear in the result")
		}
	}

	if mock.calls != 1 {
		t.Fatalf("expected exactly 1 call to repo, got %d (single round-trip required)", mock.calls)
	}
}

// TestGetPeopleInUnit_EmptyUnit verifica que una unidad sin owners ni
// occupants devuelve lista vacia (no nil) y sin error.
func TestGetPeopleInUnit_EmptyUnit(t *testing.T) {
	mock := newPeopleRepoMock()
	uc := usecases.NewGetPeopleInUnitUseCase(usecases.GetPeopleInUnitDeps{People: mock})

	resp, err := uc.Execute(context.Background(), "unit-empty")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.People == nil {
		t.Fatalf("People must be a non-nil empty slice")
	}
	if len(resp.People) != 0 {
		t.Fatalf("expected empty slice, got %d", len(resp.People))
	}
}

// TestGetPeopleInUnit_InvalidInput rechaza unit_id vacio.
func TestGetPeopleInUnit_InvalidInput(t *testing.T) {
	mock := newPeopleRepoMock()
	uc := usecases.NewGetPeopleInUnitUseCase(usecases.GetPeopleInUnitDeps{People: mock})

	_, err := uc.Execute(context.Background(), "  ")
	if !errors.Is(err, usecases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if mock.calls != 0 {
		t.Fatalf("repo must not be hit on invalid input")
	}
}

// TestGetPeopleInUnit_RepoError mapea errores de unidad inexistente.
func TestGetPeopleInUnit_NotFound(t *testing.T) {
	mock := newPeopleRepoMock()
	mock.err = domain.ErrUnitNotFound
	uc := usecases.NewGetPeopleInUnitUseCase(usecases.GetPeopleInUnitDeps{People: mock})

	_, err := uc.Execute(context.Background(), "missing")
	if !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
