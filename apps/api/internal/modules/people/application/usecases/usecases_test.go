package usecases_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/people/application/usecases"
	"github.com/saas-ph/api/internal/modules/people/domain"
	"github.com/saas-ph/api/internal/modules/people/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// --- mocks ---

type fakeVehicleRepo struct {
	createFn     func(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error)
	getByIDFn    func(ctx context.Context, id string) (entities.Vehicle, error)
	getByPlateFn func(ctx context.Context, plate string) (entities.Vehicle, error)
	listAllFn    func(ctx context.Context) ([]entities.Vehicle, error)
}

func (f *fakeVehicleRepo) Create(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error) {
	return f.createFn(ctx, in)
}
func (f *fakeVehicleRepo) GetByID(ctx context.Context, id string) (entities.Vehicle, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeVehicleRepo) GetByPlate(ctx context.Context, plate string) (entities.Vehicle, error) {
	return f.getByPlateFn(ctx, plate)
}
func (f *fakeVehicleRepo) ListAll(ctx context.Context) ([]entities.Vehicle, error) {
	return f.listAllFn(ctx)
}

type fakeAssignmentRepo struct {
	assignFn     func(ctx context.Context, in domain.AssignInput) (entities.UnitVehicleAssignment, error)
	listActiveFn func(ctx context.Context, unitID string) ([]entities.UnitVehicleAssignment, error)
	endFn        func(ctx context.Context, in domain.EndAssignmentInput) (entities.UnitVehicleAssignment, error)
}

func (f *fakeAssignmentRepo) Assign(ctx context.Context, in domain.AssignInput) (entities.UnitVehicleAssignment, error) {
	return f.assignFn(ctx, in)
}
func (f *fakeAssignmentRepo) ListActiveByUnit(ctx context.Context, unitID string) ([]entities.UnitVehicleAssignment, error) {
	return f.listActiveFn(ctx, unitID)
}
func (f *fakeAssignmentRepo) End(ctx context.Context, in domain.EndAssignmentInput) (entities.UnitVehicleAssignment, error) {
	return f.endFn(ctx, in)
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

const validUUID = "11111111-2222-3333-4444-555555555555"

// --- CreateVehicle ---

func TestCreateVehicle_Golden_NormalizesPlate(t *testing.T) {
	var captured domain.CreateVehicleInput
	repo := &fakeVehicleRepo{
		createFn: func(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error) {
			captured = in
			return entities.Vehicle{
				ID:        "11111111-2222-3333-4444-555555555555",
				Plate:     in.Plate,
				Type:      in.Type,
				Status:    entities.VehicleStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Version:   1,
			}, nil
		},
	}

	uc := usecases.CreateVehicle{Repo: repo}
	got, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "  abc123  ", // mixed case + spaces
		Type:  "car",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Plate != "ABC123" {
		t.Errorf("expected normalized plate ABC123, got %q", captured.Plate)
	}
	if captured.Type != entities.VehicleTypeCar {
		t.Errorf("expected type car, got %q", captured.Type)
	}
	if got.Plate != "ABC123" {
		t.Errorf("response plate: got %q, want ABC123", got.Plate)
	}
	if got.Status != entities.VehicleStatusActive {
		t.Errorf("expected status active, got %q", got.Status)
	}
}

func TestCreateVehicle_Golden_MotorcyclePlate(t *testing.T) {
	repo := &fakeVehicleRepo{
		createFn: func(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error) {
			return entities.Vehicle{Plate: in.Plate, Type: in.Type, Status: entities.VehicleStatusActive, Version: 1}, nil
		},
	}
	uc := usecases.CreateVehicle{Repo: repo}
	got, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "abc12a",
		Type:  "motorcycle",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Plate != "ABC12A" {
		t.Errorf("expected ABC12A, got %q", got.Plate)
	}
}

func TestCreateVehicle_PlateAlreadyExists(t *testing.T) {
	repo := &fakeVehicleRepo{
		createFn: func(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error) {
			return entities.Vehicle{}, domain.ErrPlateAlreadyExists
		},
	}
	uc := usecases.CreateVehicle{Repo: repo}
	_, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "ABC123",
		Type:  "car",
	})
	mustProblem(t, err, http.StatusConflict)
}

func TestCreateVehicle_BadPlate(t *testing.T) {
	uc := usecases.CreateVehicle{Repo: &fakeVehicleRepo{}}
	_, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "BAD",
		Type:  "car",
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateVehicle_BadType(t *testing.T) {
	uc := usecases.CreateVehicle{Repo: &fakeVehicleRepo{}}
	_, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "ABC123",
		Type:  "spaceship",
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateVehicle_BadYear(t *testing.T) {
	uc := usecases.CreateVehicle{Repo: &fakeVehicleRepo{}}
	bad := int32(1800)
	_, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "ABC123",
		Type:  "car",
		Year:  &bad,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateVehicle_RepoError(t *testing.T) {
	repo := &fakeVehicleRepo{
		createFn: func(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error) {
			return entities.Vehicle{}, errors.New("boom")
		},
	}
	uc := usecases.CreateVehicle{Repo: repo}
	_, err := uc.Execute(context.Background(), usecases.CreateVehicleInput{
		Plate: "ABC123",
		Type:  "car",
	})
	mustProblem(t, err, http.StatusInternalServerError)
}

// --- GetVehicle ---

func TestGetVehicle_BadID(t *testing.T) {
	uc := usecases.GetVehicle{Repo: &fakeVehicleRepo{}}
	_, err := uc.Execute(context.Background(), "not-a-uuid")
	mustProblem(t, err, http.StatusBadRequest)
}

func TestGetVehicle_NotFound(t *testing.T) {
	repo := &fakeVehicleRepo{
		getByIDFn: func(ctx context.Context, id string) (entities.Vehicle, error) {
			return entities.Vehicle{}, domain.ErrVehicleNotFound
		},
	}
	uc := usecases.GetVehicle{Repo: repo}
	_, err := uc.Execute(context.Background(), validUUID)
	mustProblem(t, err, http.StatusNotFound)
}

func TestGetVehicle_OK(t *testing.T) {
	want := entities.Vehicle{ID: validUUID, Plate: "ABC123", Type: entities.VehicleTypeCar, Status: entities.VehicleStatusActive, Version: 1}
	repo := &fakeVehicleRepo{
		getByIDFn: func(ctx context.Context, id string) (entities.Vehicle, error) { return want, nil },
	}
	uc := usecases.GetVehicle{Repo: repo}
	got, err := uc.Execute(context.Background(), validUUID)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("got %q want %q", got.ID, want.ID)
	}
}

// --- GetVehicleByPlate ---

func TestGetVehicleByPlate_NormalizesAndDelegates(t *testing.T) {
	var captured string
	repo := &fakeVehicleRepo{
		getByPlateFn: func(ctx context.Context, plate string) (entities.Vehicle, error) {
			captured = plate
			return entities.Vehicle{Plate: plate, Type: entities.VehicleTypeCar, Status: entities.VehicleStatusActive}, nil
		},
	}
	uc := usecases.GetVehicleByPlate{Repo: repo}
	if _, err := uc.Execute(context.Background(), "  abc123  "); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if captured != "ABC123" {
		t.Errorf("expected repo to be called with normalized plate, got %q", captured)
	}
}

func TestGetVehicleByPlate_NotFound(t *testing.T) {
	repo := &fakeVehicleRepo{
		getByPlateFn: func(ctx context.Context, plate string) (entities.Vehicle, error) {
			return entities.Vehicle{}, domain.ErrVehicleNotFound
		},
	}
	uc := usecases.GetVehicleByPlate{Repo: repo}
	_, err := uc.Execute(context.Background(), "ABC123")
	mustProblem(t, err, http.StatusNotFound)
}

// --- AssignVehicleToUnit ---

func TestAssignVehicle_BadUnitID(t *testing.T) {
	uc := usecases.AssignVehicleToUnit{Repo: &fakeAssignmentRepo{}}
	_, err := uc.Execute(context.Background(), usecases.AssignVehicleInput{
		UnitID:    "not-uuid",
		VehicleID: validUUID,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestAssignVehicle_AlreadyAssigned(t *testing.T) {
	repo := &fakeAssignmentRepo{
		assignFn: func(ctx context.Context, in domain.AssignInput) (entities.UnitVehicleAssignment, error) {
			return entities.UnitVehicleAssignment{}, domain.ErrVehicleAlreadyAssigned
		},
	}
	uc := usecases.AssignVehicleToUnit{Repo: repo}
	_, err := uc.Execute(context.Background(), usecases.AssignVehicleInput{
		UnitID: validUUID, VehicleID: validUUID,
	})
	mustProblem(t, err, http.StatusConflict)
}

func TestAssignVehicle_OK(t *testing.T) {
	repo := &fakeAssignmentRepo{
		assignFn: func(ctx context.Context, in domain.AssignInput) (entities.UnitVehicleAssignment, error) {
			return entities.UnitVehicleAssignment{
				ID:        validUUID,
				UnitID:    in.UnitID,
				VehicleID: in.VehicleID,
				Status:    entities.AssignmentStatusActive,
				Version:   1,
			}, nil
		},
	}
	uc := usecases.AssignVehicleToUnit{Repo: repo}
	got, err := uc.Execute(context.Background(), usecases.AssignVehicleInput{
		UnitID: validUUID, VehicleID: validUUID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.UnitID != validUUID || got.VehicleID != validUUID {
		t.Errorf("forwarding failed: %+v", got)
	}
}

// --- ListActiveVehiclesForUnit ---

func TestListActiveVehiclesForUnit_BadID(t *testing.T) {
	uc := usecases.ListActiveVehiclesForUnit{Repo: &fakeAssignmentRepo{}}
	_, err := uc.Execute(context.Background(), "not-uuid")
	mustProblem(t, err, http.StatusBadRequest)
}

func TestListActiveVehiclesForUnit_OK(t *testing.T) {
	repo := &fakeAssignmentRepo{
		listActiveFn: func(ctx context.Context, unitID string) ([]entities.UnitVehicleAssignment, error) {
			return []entities.UnitVehicleAssignment{{ID: validUUID, UnitID: unitID}}, nil
		},
	}
	uc := usecases.ListActiveVehiclesForUnit{Repo: repo}
	out, err := uc.Execute(context.Background(), validUUID)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(out) != 1 || out[0].UnitID != validUUID {
		t.Errorf("unexpected out: %+v", out)
	}
}

// --- EndAssignment ---

func TestEndAssignment_NotFound(t *testing.T) {
	repo := &fakeAssignmentRepo{
		endFn: func(ctx context.Context, in domain.EndAssignmentInput) (entities.UnitVehicleAssignment, error) {
			return entities.UnitVehicleAssignment{}, domain.ErrAssignmentNotFound
		},
	}
	uc := usecases.EndAssignment{Repo: repo}
	_, err := uc.Execute(context.Background(), usecases.EndAssignmentInput{AssignmentID: validUUID})
	mustProblem(t, err, http.StatusNotFound)
}

func TestEndAssignment_BadID(t *testing.T) {
	uc := usecases.EndAssignment{Repo: &fakeAssignmentRepo{}}
	_, err := uc.Execute(context.Background(), usecases.EndAssignmentInput{AssignmentID: "x"})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestEndAssignment_OK(t *testing.T) {
	repo := &fakeAssignmentRepo{
		endFn: func(ctx context.Context, in domain.EndAssignmentInput) (entities.UnitVehicleAssignment, error) {
			now := time.Now()
			return entities.UnitVehicleAssignment{
				ID:        in.AssignmentID,
				UntilDate: &now,
				Version:   2,
			}, nil
		},
	}
	uc := usecases.EndAssignment{Repo: repo}
	got, err := uc.Execute(context.Background(), usecases.EndAssignmentInput{AssignmentID: validUUID})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.UntilDate == nil {
		t.Error("expected UntilDate set")
	}
}
