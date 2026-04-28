package usecases

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	"github.com/saas-ph/api/internal/modules/packages/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ReturnPackage marca un paquete como devuelto al transportador.
//
// Pasos:
//  1. Carga el paquete y valida transicion legal (received -> returned).
//  2. DENTRO DE UNA TX: UPDATE optimista a 'returned' + outbox event.
type ReturnPackage struct {
	Packages domain.PackageRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
}

// ReturnPackageInput es el input del usecase.
type ReturnPackageInput struct {
	PackageID string
	GuardID   string
	Notes     *string
}

// Execute valida y delega.
func (u ReturnPackage) Execute(ctx context.Context, in ReturnPackageInput) (entities.Package, error) {
	if err := policies.ValidateUUID(in.PackageID); err != nil {
		return entities.Package{}, apperrors.BadRequest("package_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.GuardID); err != nil {
		return entities.Package{}, apperrors.BadRequest("guard_id: " + err.Error())
	}

	pkg, err := u.Packages.GetByID(ctx, in.PackageID)
	if err != nil {
		if errors.Is(err, domain.ErrPackageNotFound) {
			return entities.Package{}, apperrors.NotFound("package not found")
		}
		return entities.Package{}, apperrors.Internal("failed to load package")
	}
	if !policies.CanTransition(pkg.Status, entities.PackageStatusReturned) {
		return entities.Package{}, mapInvalidTransition(pkg.Status, entities.PackageStatusReturned)
	}

	var updated entities.Package
	run := func(txCtx context.Context) error {
		out, err := u.Packages.Return(txCtx, pkg.ID, pkg.Version, in.GuardID)
		if err != nil {
			return err
		}
		payload, perr := json.Marshal(map[string]any{
			"package_id":  out.ID,
			"unit_id":     out.UnitID,
			"returned_at": out.ReturnedAt,
		})
		if perr != nil {
			return perr
		}
		if _, err := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PackageID: out.ID,
			EventType: entities.OutboxEventType("package.returned"),
			Payload:   payload,
		}); err != nil {
			return err
		}
		updated = out
		return nil
	}

	exec := func(c context.Context) error {
		if u.TxRunner != nil {
			return u.TxRunner.RunInTx(c, pgx.ReadCommitted, run)
		}
		return run(c)
	}
	if err := exec(ctx); err != nil {
		switch {
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.Package{}, mapVersionConflict()
		default:
			return entities.Package{}, apperrors.Internal("failed to return package")
		}
	}
	return updated, nil
}
