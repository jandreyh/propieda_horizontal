// Package jobs contiene workers/crons del modulo packages.
//
// El worker outbox bloquea eventos pendientes con FOR UPDATE SKIP LOCKED
// y los procesa en lotes. El "envio" en MVP es solo un log; la
// integracion con notificaciones reales (push, email) se cablea despues.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/packages/application/usecases"
	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
)

// OutboxDispatcher es el "envio" de un evento outbox. La implementacion
// real (push, email, webhook) se inyecta. Devuelve nil = entregado;
// error = fallo (worker reagenda con backoff).
type OutboxDispatcher interface {
	Dispatch(ctx context.Context, event entities.OutboxEvent) error
}

// LogDispatcher es la implementacion default (MVP): solo loggea.
type LogDispatcher struct {
	Logger *slog.Logger
}

// Dispatch implementa OutboxDispatcher.
func (d LogDispatcher) Dispatch(ctx context.Context, event entities.OutboxEvent) error {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "packages.outbox: dispatch (mock)",
		slog.String("event_id", event.ID),
		slog.String("package_id", event.PackageID),
		slog.String("event_type", string(event.EventType)))
	return nil
}

// Deps agrupa las dependencias del worker.
type Deps struct {
	Logger     *slog.Logger
	Outbox     domain.OutboxRepository
	TxRunner   usecases.TxRunner
	Dispatcher OutboxDispatcher
	// BatchSize es el limite del lote en cada Tick (default 50).
	BatchSize int32
	// Interval es la cadencia de Run (default 5s).
	Interval time.Duration
}

// Worker procesa el outbox del modulo packages.
type Worker struct {
	deps Deps
}

// New construye el Worker con defaults sanos.
func New(deps Deps) *Worker {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.BatchSize <= 0 {
		deps.BatchSize = 50
	}
	if deps.Interval <= 0 {
		deps.Interval = 5 * time.Second
	}
	if deps.Dispatcher == nil {
		deps.Dispatcher = LogDispatcher{Logger: deps.Logger}
	}
	return &Worker{deps: deps}
}

// Run hace ticks cada Interval hasta que ctx se cancele.
func (w *Worker) Run(ctx context.Context) error {
	t := time.NewTicker(w.deps.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := w.Tick(ctx); err != nil {
				w.deps.Logger.ErrorContext(ctx, "packages.outbox: tick error",
					slog.String("err", err.Error()))
			}
		}
	}
}

// Tick procesa un lote de eventos pendientes dentro de UNA transaccion.
//
// Pasos:
//  1. LockPendingOutboxEvents (FOR UPDATE SKIP LOCKED, LIMIT BatchSize).
//  2. Por cada evento: dispatcher.Dispatch.
//  3. OK -> MarkDelivered. Fail -> MarkFailed con backoff exponencial.
func (w *Worker) Tick(ctx context.Context) error {
	if w.deps.TxRunner == nil {
		// Sin transaccion no podemos garantizar el lock seguro; log y salida
		// limpia.
		w.deps.Logger.WarnContext(ctx, "packages.outbox: no TxRunner; skipping tick")
		return nil
	}
	return w.deps.TxRunner.RunInTx(ctx, pgx.ReadCommitted, func(txCtx context.Context) error {
		events, err := w.deps.Outbox.LockPending(txCtx, w.deps.BatchSize)
		if err != nil {
			return err
		}
		for _, ev := range events {
			if dErr := w.deps.Dispatcher.Dispatch(txCtx, ev); dErr != nil {
				delta := backoffSeconds(ev.Attempts)
				if mErr := w.deps.Outbox.MarkFailed(txCtx, ev.ID, dErr.Error(), delta); mErr != nil {
					return mErr
				}
				continue
			}
			if mErr := w.deps.Outbox.MarkDelivered(txCtx, ev.ID); mErr != nil {
				return mErr
			}
		}
		return nil
	})
}

// backoffSeconds devuelve el delta para next_attempt_at en segundos.
// Pasos: 5s -> 15s -> 45s (luego se mantiene 45s).
func backoffSeconds(attempts int32) int {
	switch {
	case attempts <= 0:
		return 5
	case attempts == 1:
		return 15
	default:
		return 45
	}
}
