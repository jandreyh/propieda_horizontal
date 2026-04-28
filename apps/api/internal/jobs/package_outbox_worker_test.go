package jobs_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/jobs"
	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
)

// --- mocks ---

type fakeOutbox struct {
	lockFn          func(context.Context, int32) ([]entities.OutboxEvent, error)
	markDeliveredFn func(context.Context, string) error
	markFailedFn    func(context.Context, string, string, int) error

	deliveredCount atomic.Int32
	failedCount    atomic.Int32
	lastBackoff    atomic.Int32
}

func (f *fakeOutbox) Enqueue(_ context.Context, _ domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	return entities.OutboxEvent{}, nil
}
func (f *fakeOutbox) LockPending(ctx context.Context, n int32) ([]entities.OutboxEvent, error) {
	return f.lockFn(ctx, n)
}
func (f *fakeOutbox) MarkDelivered(ctx context.Context, id string) error {
	f.deliveredCount.Add(1)
	if f.markDeliveredFn != nil {
		return f.markDeliveredFn(ctx, id)
	}
	return nil
}
func (f *fakeOutbox) MarkFailed(ctx context.Context, id, e string, d int) error {
	f.failedCount.Add(1)
	f.lastBackoff.Store(int32(d))
	if f.markFailedFn != nil {
		return f.markFailedFn(ctx, id, e, d)
	}
	return nil
}

// noTxRunner ejecuta fn directamente (sin tx real) para tests.
type noTxRunner struct{}

func (noTxRunner) RunInTx(ctx context.Context, _ pgx.TxIsoLevel, fn func(context.Context) error) error {
	return fn(ctx)
}

type okDispatcher struct{}

func (okDispatcher) Dispatch(_ context.Context, _ entities.OutboxEvent) error { return nil }

type failDispatcher struct{ msg string }

func (d failDispatcher) Dispatch(_ context.Context, _ entities.OutboxEvent) error {
	return errors.New(d.msg)
}

// TestWorker_Tick_AllOk verifica que con dispatcher OK todos quedan
// marcados como delivered.
func TestWorker_Tick_AllOk(t *testing.T) {
	out := &fakeOutbox{
		lockFn: func(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
			return []entities.OutboxEvent{
				{ID: "1", PackageID: "p1", EventType: entities.OutboxEventPackageReceived},
				{ID: "2", PackageID: "p2", EventType: entities.OutboxEventPackageDelivered},
			}, nil
		},
	}
	w := jobs.New(jobs.Deps{
		Outbox:     out,
		TxRunner:   noTxRunner{},
		Dispatcher: okDispatcher{},
	})
	if err := w.Tick(context.Background()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := out.deliveredCount.Load(); got != 2 {
		t.Errorf("expected 2 delivered, got %d", got)
	}
	if got := out.failedCount.Load(); got != 0 {
		t.Errorf("expected 0 failed, got %d", got)
	}
}

// TestWorker_Tick_DispatcherFails verifica que cuando el dispatcher
// falla, los eventos quedan marcados como failed con backoff.
func TestWorker_Tick_DispatcherFails(t *testing.T) {
	out := &fakeOutbox{
		lockFn: func(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
			return []entities.OutboxEvent{
				{ID: "1", PackageID: "p1", EventType: entities.OutboxEventPackageReceived, Attempts: 0},
				{ID: "2", PackageID: "p2", EventType: entities.OutboxEventPackageReceived, Attempts: 1},
				{ID: "3", PackageID: "p3", EventType: entities.OutboxEventPackageReceived, Attempts: 5},
			}, nil
		},
	}
	w := jobs.New(jobs.Deps{
		Outbox:     out,
		TxRunner:   noTxRunner{},
		Dispatcher: failDispatcher{msg: "boom"},
	})
	if err := w.Tick(context.Background()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := out.failedCount.Load(); got != 3 {
		t.Errorf("expected 3 failed, got %d", got)
	}
	// El ultimo fallo (attempts=5) debe usar el plateau: 45s.
	if got := out.lastBackoff.Load(); got != 45 {
		t.Errorf("expected backoff=45 for attempts>=2, got %d", got)
	}
}

// TestWorker_Tick_NoTxRunner es no-op (no panico).
func TestWorker_Tick_NoTxRunner(t *testing.T) {
	out := &fakeOutbox{
		lockFn: func(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
			t.Error("LockPending should not be called without TxRunner")
			return nil, nil
		},
	}
	w := jobs.New(jobs.Deps{Outbox: out})
	if err := w.Tick(context.Background()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

// TestWorker_Run_RespectsContextCancel verifica que Run sale cuando
// ctx se cancela.
func TestWorker_Run_RespectsContextCancel(t *testing.T) {
	out := &fakeOutbox{
		lockFn: func(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
			return nil, nil
		},
	}
	w := jobs.New(jobs.Deps{
		Outbox:     out,
		TxRunner:   noTxRunner{},
		Dispatcher: okDispatcher{},
		Interval:   10 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := w.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

// --- ReminderCron ---

type fakeReminderPackages struct {
	listFn func(context.Context) ([]entities.Package, error)
}

func (f *fakeReminderPackages) Create(_ context.Context, _ domain.CreatePackageInput) (entities.Package, error) {
	return entities.Package{}, nil
}
func (f *fakeReminderPackages) GetByID(_ context.Context, _ string) (entities.Package, error) {
	return entities.Package{}, nil
}
func (f *fakeReminderPackages) ListByUnit(_ context.Context, _ string) ([]entities.Package, error) {
	return nil, nil
}
func (f *fakeReminderPackages) ListByStatus(_ context.Context, _ entities.PackageStatus) ([]entities.Package, error) {
	return nil, nil
}
func (f *fakeReminderPackages) UpdateStatusOptimistic(_ context.Context, _ string, _ int32, _ entities.PackageStatus, _ string) (entities.Package, error) {
	return entities.Package{}, nil
}
func (f *fakeReminderPackages) Return(_ context.Context, _ string, _ int32, _ string) (entities.Package, error) {
	return entities.Package{}, nil
}
func (f *fakeReminderPackages) ListPendingReminder(ctx context.Context) ([]entities.Package, error) {
	return f.listFn(ctx)
}

type fakeReminderOutbox struct {
	enqueueCount atomic.Int32
}

func (f *fakeReminderOutbox) Enqueue(_ context.Context, in domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	f.enqueueCount.Add(1)
	return entities.OutboxEvent{ID: "x", PackageID: in.PackageID, EventType: in.EventType}, nil
}
func (f *fakeReminderOutbox) LockPending(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeReminderOutbox) MarkDelivered(_ context.Context, _ string) error { return nil }
func (f *fakeReminderOutbox) MarkFailed(_ context.Context, _, _ string, _ int) error {
	return nil
}

func TestReminderCron_Tick_EnqueuesOnePerPackage(t *testing.T) {
	pkgs := &fakeReminderPackages{
		listFn: func(_ context.Context) ([]entities.Package, error) {
			return []entities.Package{
				{ID: "p1", Status: entities.PackageStatusReceived},
				{ID: "p2", Status: entities.PackageStatusReceived},
				{ID: "p3", Status: entities.PackageStatusReceived},
			}, nil
		},
	}
	out := &fakeReminderOutbox{}
	c := jobs.NewReminderCron(jobs.ReminderDeps{Packages: pkgs, Outbox: out})
	if err := c.Tick(context.Background()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := out.enqueueCount.Load(); got != 3 {
		t.Errorf("expected 3 enqueues, got %d", got)
	}
}
