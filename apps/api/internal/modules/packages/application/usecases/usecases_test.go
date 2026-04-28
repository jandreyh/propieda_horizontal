package usecases_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/packages/application/usecases"
	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// --- mocks ---

type fakePackages struct {
	createFn       func(context.Context, domain.CreatePackageInput) (entities.Package, error)
	getByIDFn      func(context.Context, string) (entities.Package, error)
	listByUnitFn   func(context.Context, string) ([]entities.Package, error)
	listByStatusFn func(context.Context, entities.PackageStatus) ([]entities.Package, error)
	updateFn       func(context.Context, string, int32, entities.PackageStatus, string) (entities.Package, error)
	returnFn       func(context.Context, string, int32, string) (entities.Package, error)
	listPendingFn  func(context.Context) ([]entities.Package, error)
}

func (f *fakePackages) Create(ctx context.Context, in domain.CreatePackageInput) (entities.Package, error) {
	return f.createFn(ctx, in)
}
func (f *fakePackages) GetByID(ctx context.Context, id string) (entities.Package, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakePackages) ListByUnit(ctx context.Context, unitID string) ([]entities.Package, error) {
	return f.listByUnitFn(ctx, unitID)
}
func (f *fakePackages) ListByStatus(ctx context.Context, s entities.PackageStatus) ([]entities.Package, error) {
	return f.listByStatusFn(ctx, s)
}
func (f *fakePackages) UpdateStatusOptimistic(ctx context.Context, id string, v int32, s entities.PackageStatus, actor string) (entities.Package, error) {
	return f.updateFn(ctx, id, v, s, actor)
}
func (f *fakePackages) Return(ctx context.Context, id string, v int32, actor string) (entities.Package, error) {
	return f.returnFn(ctx, id, v, actor)
}
func (f *fakePackages) ListPendingReminder(ctx context.Context) ([]entities.Package, error) {
	return f.listPendingFn(ctx)
}

type fakeCategories struct {
	listFn      func(context.Context) ([]entities.PackageCategory, error)
	getByNameFn func(context.Context, string) (entities.PackageCategory, error)
	getByIDFn   func(context.Context, string) (entities.PackageCategory, error)
}

func (f *fakeCategories) List(ctx context.Context) ([]entities.PackageCategory, error) {
	return f.listFn(ctx)
}
func (f *fakeCategories) GetByName(ctx context.Context, n string) (entities.PackageCategory, error) {
	return f.getByNameFn(ctx, n)
}
func (f *fakeCategories) GetByID(ctx context.Context, id string) (entities.PackageCategory, error) {
	return f.getByIDFn(ctx, id)
}

type fakeDeliveries struct {
	recordFn func(context.Context, domain.RecordDeliveryInput) (entities.DeliveryEvent, error)
}

func (f *fakeDeliveries) Record(ctx context.Context, in domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
	return f.recordFn(ctx, in)
}

type fakeOutbox struct {
	enqueueFn       func(context.Context, domain.EnqueueOutboxInput) (entities.OutboxEvent, error)
	lockPendingFn   func(context.Context, int32) ([]entities.OutboxEvent, error)
	markDeliveredFn func(context.Context, string) error
	markFailedFn    func(context.Context, string, string, int) error
	enqueueCount    atomic.Int32
}

func (f *fakeOutbox) Enqueue(ctx context.Context, in domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	f.enqueueCount.Add(1)
	if f.enqueueFn == nil {
		return entities.OutboxEvent{ID: "outbox-id", PackageID: in.PackageID, EventType: in.EventType}, nil
	}
	return f.enqueueFn(ctx, in)
}
func (f *fakeOutbox) LockPending(ctx context.Context, n int32) ([]entities.OutboxEvent, error) {
	return f.lockPendingFn(ctx, n)
}
func (f *fakeOutbox) MarkDelivered(ctx context.Context, id string) error {
	return f.markDeliveredFn(ctx, id)
}
func (f *fakeOutbox) MarkFailed(ctx context.Context, id, e string, d int) error {
	return f.markFailedFn(ctx, id, e, d)
}

// --- helpers ---

const validPackageID = "11111111-2222-3333-4444-555555555555"
const validUnitID = "22222222-3333-4444-5555-666666666666"
const validUserID = "33333333-4444-5555-6666-777777777777"
const validGuardID = "44444444-5555-6666-7777-888888888888"

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

// --- CreatePackage ---

func TestCreatePackage_Golden(t *testing.T) {
	var captured domain.CreatePackageInput
	pkgs := &fakePackages{
		createFn: func(ctx context.Context, in domain.CreatePackageInput) (entities.Package, error) {
			captured = in
			return entities.Package{
				ID:               validPackageID,
				UnitID:           in.UnitID,
				RecipientName:    in.RecipientName,
				ReceivedByUserID: in.ReceivedByUserID,
				Status:           entities.PackageStatusReceived,
				Version:          1,
			}, nil
		},
	}
	out := &fakeOutbox{}
	cats := &fakeCategories{}
	uc := usecases.CreatePackage{Packages: pkgs, Categories: cats, Outbox: out}
	res, err := uc.Execute(context.Background(), usecases.CreatePackageInput{
		UnitID:           validUnitID,
		RecipientName:    "Juan Perez",
		ReceivedByUserID: validUserID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res.Status != entities.PackageStatusReceived {
		t.Errorf("expected received, got %s", res.Status)
	}
	if captured.UnitID != validUnitID {
		t.Errorf("captured.UnitID = %q", captured.UnitID)
	}
	if out.enqueueCount.Load() != 1 {
		t.Errorf("expected 1 outbox enqueue, got %d", out.enqueueCount.Load())
	}
}

func TestCreatePackage_BadUnitID(t *testing.T) {
	uc := usecases.CreatePackage{Packages: &fakePackages{}, Categories: &fakeCategories{}, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.CreatePackageInput{
		UnitID:           "bad",
		RecipientName:    "X",
		ReceivedByUserID: validUserID,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreatePackage_EvidenceRequired(t *testing.T) {
	categoryID := "55555555-6666-7777-8888-999999999999"
	cats := &fakeCategories{
		getByIDFn: func(_ context.Context, id string) (entities.PackageCategory, error) {
			return entities.PackageCategory{
				ID:               id,
				Name:             "Refrigerado",
				RequiresEvidence: true,
			}, nil
		},
	}
	uc := usecases.CreatePackage{Packages: &fakePackages{}, Categories: cats, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.CreatePackageInput{
		UnitID:           validUnitID,
		RecipientName:    "Juan",
		ReceivedByUserID: validUserID,
		CategoryID:       &categoryID,
		// sin ReceivedEvidenceURL
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreatePackage_EvidenceProvided_OK(t *testing.T) {
	categoryID := "55555555-6666-7777-8888-999999999999"
	url := "https://x/photo.jpg"
	cats := &fakeCategories{
		getByIDFn: func(_ context.Context, id string) (entities.PackageCategory, error) {
			return entities.PackageCategory{
				ID:               id,
				Name:             "Refrigerado",
				RequiresEvidence: true,
			}, nil
		},
	}
	pkgs := &fakePackages{
		createFn: func(_ context.Context, in domain.CreatePackageInput) (entities.Package, error) {
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusReceived}, nil
		},
	}
	uc := usecases.CreatePackage{Packages: pkgs, Categories: cats, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.CreatePackageInput{
		UnitID:              validUnitID,
		RecipientName:       "Juan",
		ReceivedByUserID:    validUserID,
		CategoryID:          &categoryID,
		ReceivedEvidenceURL: &url,
	})
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

// --- DeliverByQR ---

func TestDeliverByQR_Golden(t *testing.T) {
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{
				ID:      validPackageID,
				UnitID:  validUnitID,
				Status:  entities.PackageStatusReceived,
				Version: 1,
			}, nil
		},
		updateFn: func(_ context.Context, id string, v int32, s entities.PackageStatus, _ string) (entities.Package, error) {
			now := time.Now()
			return entities.Package{
				ID:          id,
				UnitID:      validUnitID,
				Status:      s,
				Version:     v + 1,
				DeliveredAt: &now,
			}, nil
		},
	}
	dels := &fakeDeliveries{
		recordFn: func(_ context.Context, in domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
			return entities.DeliveryEvent{
				ID:                "ev-id",
				PackageID:         in.PackageID,
				DeliveryMethod:    in.DeliveryMethod,
				DeliveredByUserID: in.DeliveredByUserID,
				Status:            entities.DeliveryEventStatusCompleted,
			}, nil
		},
	}
	out := &fakeOutbox{}
	uc := usecases.DeliverByQR{Packages: pkgs, Deliveries: dels, Outbox: out}
	res, err := uc.Execute(context.Background(), usecases.DeliverByQRInput{
		PackageID:         validPackageID,
		DeliveredToUserID: validUserID,
		GuardID:           validGuardID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res.Package.Status != entities.PackageStatusDelivered {
		t.Errorf("expected delivered, got %s", res.Package.Status)
	}
	if res.Event.DeliveryMethod != entities.DeliveryMethodQR {
		t.Errorf("expected qr method, got %s", res.Event.DeliveryMethod)
	}
	if out.enqueueCount.Load() != 1 {
		t.Errorf("expected 1 outbox enqueue, got %d", out.enqueueCount.Load())
	}
}

func TestDeliverByQR_AlreadyDelivered_409(t *testing.T) {
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{
				ID:      validPackageID,
				Status:  entities.PackageStatusDelivered,
				Version: 2,
			}, nil
		},
	}
	uc := usecases.DeliverByQR{Packages: pkgs, Deliveries: &fakeDeliveries{}, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.DeliverByQRInput{
		PackageID:         validPackageID,
		DeliveredToUserID: validUserID,
		GuardID:           validGuardID,
	})
	mustProblem(t, err, http.StatusConflict)
}

func TestDeliverByQR_VersionConflict_409(t *testing.T) {
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusReceived, Version: 1}, nil
		},
		updateFn: func(_ context.Context, _ string, _ int32, _ entities.PackageStatus, _ string) (entities.Package, error) {
			return entities.Package{}, domain.ErrVersionConflict
		},
	}
	uc := usecases.DeliverByQR{Packages: pkgs, Deliveries: &fakeDeliveries{}, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.DeliverByQRInput{
		PackageID:         validPackageID,
		DeliveredToUserID: validUserID,
		GuardID:           validGuardID,
	})
	mustProblem(t, err, http.StatusConflict)
}

func TestDeliverByQR_Idempotency_Hit(t *testing.T) {
	calls := atomic.Int32{}
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			calls.Add(1)
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusReceived, Version: 1}, nil
		},
		updateFn: func(_ context.Context, id string, v int32, s entities.PackageStatus, _ string) (entities.Package, error) {
			return entities.Package{ID: id, Status: s, Version: v + 1}, nil
		},
	}
	dels := &fakeDeliveries{
		recordFn: func(_ context.Context, in domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
			return entities.DeliveryEvent{ID: "ev1", PackageID: in.PackageID, DeliveryMethod: in.DeliveryMethod}, nil
		},
	}
	cache := usecases.NewIdempotencyCache(time.Hour, time.Now)
	uc := usecases.DeliverByQR{Packages: pkgs, Deliveries: dels, Outbox: &fakeOutbox{}, Idempotency: cache}
	in := usecases.DeliverByQRInput{
		PackageID:         validPackageID,
		DeliveredToUserID: validUserID,
		GuardID:           validGuardID,
		IdempotencyKey:    "key-123",
	}
	r1, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	r2, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if r1.Package.ID != r2.Package.ID {
		t.Errorf("expected same package id, got %s vs %s", r1.Package.ID, r2.Package.ID)
	}
	// Solo la primera llamada debe haber tocado el repo.
	if calls.Load() != 1 {
		t.Errorf("expected 1 GetByID call, got %d", calls.Load())
	}
}

// --- DeliverManual ---

func TestDeliverManual_NoEvidence_400(t *testing.T) {
	uc := usecases.DeliverManual{Packages: &fakePackages{}, Deliveries: &fakeDeliveries{}, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.DeliverManualInput{
		PackageID: validPackageID,
		GuardID:   validGuardID,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestDeliverManual_WithSignature_OK(t *testing.T) {
	sig := "https://x/sig.png"
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusReceived, Version: 1}, nil
		},
		updateFn: func(_ context.Context, id string, v int32, s entities.PackageStatus, _ string) (entities.Package, error) {
			return entities.Package{ID: id, Status: s, Version: v + 1}, nil
		},
	}
	dels := &fakeDeliveries{
		recordFn: func(_ context.Context, in domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
			return entities.DeliveryEvent{ID: "ev2", PackageID: in.PackageID, DeliveryMethod: in.DeliveryMethod}, nil
		},
	}
	uc := usecases.DeliverManual{Packages: pkgs, Deliveries: dels, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.DeliverManualInput{
		PackageID:    validPackageID,
		SignatureURL: &sig,
		GuardID:      validGuardID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestDeliverManual_AlreadyReturned_409(t *testing.T) {
	sig := "https://x/sig.png"
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusReturned, Version: 1}, nil
		},
	}
	uc := usecases.DeliverManual{Packages: pkgs, Deliveries: &fakeDeliveries{}, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.DeliverManualInput{
		PackageID:    validPackageID,
		SignatureURL: &sig,
		GuardID:      validGuardID,
	})
	mustProblem(t, err, http.StatusConflict)
}

// --- Concurrencia: 2 goroutines llaman DeliverByQR simultaneamente ---
//
// El mock simula la condicion de carrera del UPDATE optimista: el primer
// UPDATE recibido devuelve OK; el segundo (que tiene la version vieja)
// devuelve ErrVersionConflict.
func TestDeliverByQR_ConcurrentDelivery_OneWinsOneConflicts(t *testing.T) {
	var (
		updMu       sync.Mutex
		updateFired atomic.Int32
	)
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{
				ID: validPackageID, Status: entities.PackageStatusReceived, Version: 1,
			}, nil
		},
		updateFn: func(_ context.Context, id string, v int32, s entities.PackageStatus, _ string) (entities.Package, error) {
			updMu.Lock()
			defer updMu.Unlock()
			n := updateFired.Add(1)
			if n == 1 {
				return entities.Package{ID: id, Status: s, Version: v + 1}, nil
			}
			return entities.Package{}, domain.ErrVersionConflict
		},
	}
	dels := &fakeDeliveries{
		recordFn: func(_ context.Context, in domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
			return entities.DeliveryEvent{ID: "ev", PackageID: in.PackageID, DeliveryMethod: in.DeliveryMethod}, nil
		},
	}
	uc := usecases.DeliverByQR{Packages: pkgs, Deliveries: dels, Outbox: &fakeOutbox{}}
	in := usecases.DeliverByQRInput{
		PackageID:         validPackageID,
		DeliveredToUserID: validUserID,
		GuardID:           validGuardID,
	}

	var wg sync.WaitGroup
	results := make([]error, 2)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := uc.Execute(context.Background(), in)
			results[idx] = err
		}(i)
	}
	wg.Wait()

	// Exactamente una respuesta nil y una 409 ErrVersionConflict mapeado.
	var oks, conflicts int
	for _, e := range results {
		if e == nil {
			oks++
			continue
		}
		var p apperrors.Problem
		if errors.As(e, &p) && p.Status == http.StatusConflict {
			conflicts++
			continue
		}
		t.Errorf("unexpected error: %v", e)
	}
	if oks != 1 || conflicts != 1 {
		t.Errorf("expected 1 ok + 1 conflict, got oks=%d conflicts=%d", oks, conflicts)
	}
}

// --- ReturnPackage ---

func TestReturnPackage_Golden(t *testing.T) {
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusReceived, Version: 1}, nil
		},
		returnFn: func(_ context.Context, id string, v int32, _ string) (entities.Package, error) {
			now := time.Now()
			return entities.Package{ID: id, Status: entities.PackageStatusReturned, Version: v + 1, ReturnedAt: &now}, nil
		},
	}
	uc := usecases.ReturnPackage{Packages: pkgs, Outbox: &fakeOutbox{}}
	out, err := uc.Execute(context.Background(), usecases.ReturnPackageInput{
		PackageID: validPackageID,
		GuardID:   validGuardID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Status != entities.PackageStatusReturned {
		t.Errorf("expected returned, got %s", out.Status)
	}
}

func TestReturnPackage_AlreadyDelivered_409(t *testing.T) {
	pkgs := &fakePackages{
		getByIDFn: func(_ context.Context, _ string) (entities.Package, error) {
			return entities.Package{ID: validPackageID, Status: entities.PackageStatusDelivered, Version: 2}, nil
		},
	}
	uc := usecases.ReturnPackage{Packages: pkgs, Outbox: &fakeOutbox{}}
	_, err := uc.Execute(context.Background(), usecases.ReturnPackageInput{
		PackageID: validPackageID,
		GuardID:   validGuardID,
	})
	mustProblem(t, err, http.StatusConflict)
}

// --- Idempotency cache ---

func TestIdempotencyCache_TTL(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	clock := now
	cache := usecases.NewIdempotencyCache(time.Minute, func() time.Time { return clock })
	cache.Set("k", "v")
	if v, ok := cache.Get("k"); !ok || v != "v" {
		t.Errorf("expected hit, got ok=%v v=%v", ok, v)
	}
	clock = now.Add(2 * time.Minute)
	if _, ok := cache.Get("k"); ok {
		t.Error("expected miss after TTL")
	}
}

func TestIdempotencyCache_EmptyKey(t *testing.T) {
	cache := usecases.NewIdempotencyCache(time.Hour, time.Now)
	cache.Set("", "v") // no-op
	if _, ok := cache.Get(""); ok {
		t.Error("empty key should never hit")
	}
}
