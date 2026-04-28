package usecases_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/residential_structure/application/usecases"
	"github.com/saas-ph/api/internal/modules/residential_structure/domain"
	"github.com/saas-ph/api/internal/modules/residential_structure/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// --- mocks ---

type fakeStructureRepo struct {
	listActiveFn   func(ctx context.Context) ([]entities.Structure, error)
	getByIDFn      func(ctx context.Context, id string) (entities.Structure, error)
	createFn       func(ctx context.Context, in domain.CreateStructureInput) (entities.Structure, error)
	updateFn       func(ctx context.Context, in domain.UpdateStructureInput) (entities.Structure, error)
	archiveFn      func(ctx context.Context, id, actor string) error
	listChildrenFn func(ctx context.Context, parentID string) ([]entities.Structure, error)
}

func (f *fakeStructureRepo) ListActive(ctx context.Context) ([]entities.Structure, error) {
	return f.listActiveFn(ctx)
}
func (f *fakeStructureRepo) GetByID(ctx context.Context, id string) (entities.Structure, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeStructureRepo) Create(ctx context.Context, in domain.CreateStructureInput) (entities.Structure, error) {
	return f.createFn(ctx, in)
}
func (f *fakeStructureRepo) Update(ctx context.Context, in domain.UpdateStructureInput) (entities.Structure, error) {
	return f.updateFn(ctx, in)
}
func (f *fakeStructureRepo) Archive(ctx context.Context, id, actor string) error {
	return f.archiveFn(ctx, id, actor)
}
func (f *fakeStructureRepo) ListChildren(ctx context.Context, parentID string) ([]entities.Structure, error) {
	return f.listChildrenFn(ctx, parentID)
}

// --- helpers ---

func mustProblem(t *testing.T, err error, status int) {
	t.Helper()
	var p apperrors.Problem
	if !errors.As(err, &p) {
		t.Fatalf("expected apperrors.Problem, got %v", err)
	}
	if p.Status != status {
		t.Fatalf("expected status %d, got %d (%s)", status, p.Status, p.Detail)
	}
}

// --- ListStructures golden path ---

func TestListStructures_OK(t *testing.T) {
	now := time.Now()
	want := []entities.Structure{
		{
			ID:         "00000000-0000-0000-0000-000000000001",
			Name:       "Torre A",
			Type:       entities.StructureTypeTower,
			OrderIndex: 1,
			Status:     entities.StructureStatusActive,
			CreatedAt:  now,
			UpdatedAt:  now,
			Version:    1,
		},
		{
			ID:         "00000000-0000-0000-0000-000000000002",
			Name:       "Torre B",
			Type:       entities.StructureTypeTower,
			OrderIndex: 2,
			Status:     entities.StructureStatusActive,
			CreatedAt:  now,
			UpdatedAt:  now,
			Version:    1,
		},
	}
	uc := usecases.ListStructures{Repo: &fakeStructureRepo{
		listActiveFn: func(ctx context.Context) ([]entities.Structure, error) {
			return want, nil
		},
	}}
	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Total != 2 {
		t.Fatalf("expected total 2, got %d", out.Total)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out.Items))
	}
	if out.Items[0].Name != "Torre A" {
		t.Fatalf("expected Torre A, got %q", out.Items[0].Name)
	}
}

func TestListStructures_RepoError(t *testing.T) {
	uc := usecases.ListStructures{Repo: &fakeStructureRepo{
		listActiveFn: func(ctx context.Context) ([]entities.Structure, error) {
			return nil, errors.New("boom")
		},
	}}
	_, err := uc.Execute(context.Background())
	mustProblem(t, err, http.StatusInternalServerError)
}

// --- CreateStructure golden path ---

func TestCreateStructure_OK(t *testing.T) {
	var captured domain.CreateStructureInput
	uc := usecases.CreateStructure{Repo: &fakeStructureRepo{
		createFn: func(ctx context.Context, in domain.CreateStructureInput) (entities.Structure, error) {
			captured = in
			return entities.Structure{
				ID:         "00000000-0000-0000-0000-000000000010",
				Name:       in.Name,
				Type:       in.Type,
				ParentID:   in.ParentID,
				OrderIndex: in.OrderIndex,
				Status:     entities.StructureStatusActive,
				Version:    1,
			}, nil
		},
	}}
	got, err := uc.Execute(context.Background(), usecases.CreateStructureInput{
		Name:       "Torre A",
		Type:       "tower",
		OrderIndex: 5,
		ActorID:    "actor-1",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.Name != "Torre A" {
		t.Fatalf("name: %q", got.Name)
	}
	if got.Type != entities.StructureTypeTower {
		t.Fatalf("type: %q", got.Type)
	}
	if got.Version != 1 {
		t.Fatalf("version: %d", got.Version)
	}
	if captured.ActorID != "actor-1" {
		t.Errorf("actor not forwarded: %q", captured.ActorID)
	}
	if captured.OrderIndex != 5 {
		t.Errorf("order_index not forwarded: %d", captured.OrderIndex)
	}
}

func TestCreateStructure_BadName(t *testing.T) {
	uc := usecases.CreateStructure{Repo: &fakeStructureRepo{}}
	_, err := uc.Execute(context.Background(), usecases.CreateStructureInput{
		Name: "  ", Type: "tower",
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateStructure_BadType(t *testing.T) {
	uc := usecases.CreateStructure{Repo: &fakeStructureRepo{}}
	_, err := uc.Execute(context.Background(), usecases.CreateStructureInput{
		Name: "Torre A", Type: "invalid",
	})
	mustProblem(t, err, http.StatusBadRequest)
}
