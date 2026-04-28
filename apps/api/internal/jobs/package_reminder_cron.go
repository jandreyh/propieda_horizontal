package jobs

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
)

// ReminderDeps agrupa las dependencias del cron de re-notificacion.
type ReminderDeps struct {
	Logger   *slog.Logger
	Packages domain.PackageRepository
	Outbox   domain.OutboxRepository
}

// ReminderCron emite eventos 'package.reminder' por cada paquete que
// lleva mas de 3 dias en estado 'received' y SIN reminder en las
// ultimas 24h. El scheduler concreto (cron diaria 8am tenant TZ) se
// cablea desde el orquestador; aqui solo expone Tick.
type ReminderCron struct {
	deps ReminderDeps
}

// NewReminderCron construye un ReminderCron.
func NewReminderCron(deps ReminderDeps) *ReminderCron {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	return &ReminderCron{deps: deps}
}

// Tick busca los paquetes pendientes de recordatorio y encola un evento
// outbox por cada uno. Es idempotente: la query excluye paquetes con
// reminder reciente.
func (c *ReminderCron) Tick(ctx context.Context) error {
	pkgs, err := c.deps.Packages.ListPendingReminder(ctx)
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		payload, perr := json.Marshal(map[string]any{
			"package_id":     p.ID,
			"unit_id":        p.UnitID,
			"recipient_name": p.RecipientName,
			"received_at":    p.ReceivedAt,
		})
		if perr != nil {
			c.deps.Logger.WarnContext(ctx, "packages.reminder: marshal error",
				slog.String("package_id", p.ID),
				slog.String("err", perr.Error()))
			continue
		}
		if _, err := c.deps.Outbox.Enqueue(ctx, domain.EnqueueOutboxInput{
			PackageID: p.ID,
			EventType: entities.OutboxEventPackageReminder,
			Payload:   payload,
		}); err != nil {
			c.deps.Logger.WarnContext(ctx, "packages.reminder: enqueue error",
				slog.String("package_id", p.ID),
				slog.String("err", err.Error()))
			continue
		}
	}
	return nil
}
