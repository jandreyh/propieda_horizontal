package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	"github.com/saas-ph/api/internal/modules/packages/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// DeliverResult es el output de un deliver-* usecase.
type DeliverResult struct {
	Package entities.Package
	Event   entities.DeliveryEvent
}

// DeliverByQR entrega un paquete por QR (residente identificado).
//
// Pasos:
//  1. Si IdempotencyKey != "" y el cache tiene la respuesta, retorna esa.
//  2. Carga el paquete (lock optimista por version).
//  3. Valida que esta en 'received' (sino ErrInvalidTransition).
//  4. DENTRO DE UNA TX (READ COMMITTED): UPDATE optimista + INSERT
//     delivery event + ENQUEUE outbox 'package.delivered'.
//  5. Cachea la respuesta bajo IdempotencyKey si vino.
type DeliverByQR struct {
	Packages    domain.PackageRepository
	Deliveries  domain.DeliveryRepository
	Outbox      domain.OutboxRepository
	TxRunner    TxRunner
	Idempotency *IdempotencyCache
}

// DeliverByQRInput es el input del usecase.
type DeliverByQRInput struct {
	PackageID         string
	DeliveredToUserID string
	GuardID           string
	IdempotencyKey    string
	Notes             *string
}

// Execute orquesta el flujo descrito en el doc del struct.
func (u DeliverByQR) Execute(ctx context.Context, in DeliverByQRInput) (DeliverResult, error) {
	if err := policies.ValidateUUID(in.PackageID); err != nil {
		return DeliverResult{}, apperrors.BadRequest("package_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.DeliveredToUserID); err != nil {
		return DeliverResult{}, apperrors.BadRequest("delivered_to_user_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.GuardID); err != nil {
		return DeliverResult{}, apperrors.BadRequest("guard_id: " + err.Error())
	}

	// Idempotency hit antes de tocar la base.
	if u.Idempotency != nil && in.IdempotencyKey != "" {
		if v, ok := u.Idempotency.Get(in.IdempotencyKey); ok {
			if r, ok := v.(DeliverResult); ok {
				return r, nil
			}
		}
	}

	pkg, err := u.Packages.GetByID(ctx, in.PackageID)
	if err != nil {
		if errors.Is(err, domain.ErrPackageNotFound) {
			return DeliverResult{}, apperrors.NotFound("package not found")
		}
		return DeliverResult{}, apperrors.Internal("failed to load package")
	}
	if !policies.CanTransition(pkg.Status, entities.PackageStatusDelivered) {
		return DeliverResult{}, mapInvalidTransition(pkg.Status, entities.PackageStatusDelivered)
	}

	deliveredTo := in.DeliveredToUserID
	out, err := u.runDelivery(ctx, deliveryRun{
		Package: pkg,
		Method:  entities.DeliveryMethodQR,
		Input: domain.RecordDeliveryInput{
			PackageID:           pkg.ID,
			DeliveredToUserID:   &deliveredTo,
			RecipientNameManual: nil,
			DeliveryMethod:      entities.DeliveryMethodQR,
			SignatureURL:        nil,
			PhotoEvidenceURL:    nil,
			DeliveredByUserID:   in.GuardID,
			Notes:               in.Notes,
		},
		ActorID: in.GuardID,
	})
	if err != nil {
		return DeliverResult{}, err
	}

	if u.Idempotency != nil && in.IdempotencyKey != "" {
		u.Idempotency.Set(in.IdempotencyKey, out)
	}
	return out, nil
}

// DeliverManual entrega un paquete con firma o foto del receptor.
//
// Reglas extra: DEBE haber al menos una de SignatureURL o
// PhotoEvidenceURL (sino 400 ErrEvidenceRequired).
type DeliverManual struct {
	Packages    domain.PackageRepository
	Deliveries  domain.DeliveryRepository
	Outbox      domain.OutboxRepository
	TxRunner    TxRunner
	Idempotency *IdempotencyCache
}

// DeliverManualInput es el input del usecase.
type DeliverManualInput struct {
	PackageID           string
	RecipientNameManual *string
	SignatureURL        *string
	PhotoEvidenceURL    *string
	GuardID             string
	IdempotencyKey      string
	Notes               *string
}

// Execute orquesta el flujo del struct.
func (u DeliverManual) Execute(ctx context.Context, in DeliverManualInput) (DeliverResult, error) {
	if err := policies.ValidateUUID(in.PackageID); err != nil {
		return DeliverResult{}, apperrors.BadRequest("package_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.GuardID); err != nil {
		return DeliverResult{}, apperrors.BadRequest("guard_id: " + err.Error())
	}
	if !hasNonEmpty(in.SignatureURL) && !hasNonEmpty(in.PhotoEvidenceURL) {
		return DeliverResult{}, mapManualEvidenceRequired()
	}

	if u.Idempotency != nil && in.IdempotencyKey != "" {
		if v, ok := u.Idempotency.Get(in.IdempotencyKey); ok {
			if r, ok := v.(DeliverResult); ok {
				return r, nil
			}
		}
	}

	pkg, err := u.Packages.GetByID(ctx, in.PackageID)
	if err != nil {
		if errors.Is(err, domain.ErrPackageNotFound) {
			return DeliverResult{}, apperrors.NotFound("package not found")
		}
		return DeliverResult{}, apperrors.Internal("failed to load package")
	}
	if !policies.CanTransition(pkg.Status, entities.PackageStatusDelivered) {
		return DeliverResult{}, mapInvalidTransition(pkg.Status, entities.PackageStatusDelivered)
	}

	out, err := u.runDelivery(ctx, deliveryRun{
		Package: pkg,
		Method:  entities.DeliveryMethodManual,
		Input: domain.RecordDeliveryInput{
			PackageID:           pkg.ID,
			DeliveredToUserID:   nil,
			RecipientNameManual: in.RecipientNameManual,
			DeliveryMethod:      entities.DeliveryMethodManual,
			SignatureURL:        in.SignatureURL,
			PhotoEvidenceURL:    in.PhotoEvidenceURL,
			DeliveredByUserID:   in.GuardID,
			Notes:               in.Notes,
		},
		ActorID: in.GuardID,
	})
	if err != nil {
		return DeliverResult{}, err
	}

	if u.Idempotency != nil && in.IdempotencyKey != "" {
		u.Idempotency.Set(in.IdempotencyKey, out)
	}
	return out, nil
}

// deliveryRun encapsula el cuerpo transaccional comun de Deliver*.
type deliveryRun struct {
	Package entities.Package
	Method  entities.DeliveryMethod
	Input   domain.RecordDeliveryInput
	ActorID string
}

// runDelivery ejecuta UPDATE optimista + INSERT delivery + ENQUEUE outbox
// dentro de una tx. Es la rutina compartida de DeliverByQR/Manual.
func (u DeliverByQR) runDelivery(ctx context.Context, dr deliveryRun) (DeliverResult, error) {
	return runDeliveryShared(ctx, runDeliveryDeps{
		Packages:   u.Packages,
		Deliveries: u.Deliveries,
		Outbox:     u.Outbox,
		TxRunner:   u.TxRunner,
	}, dr)
}

// runDelivery (manual) usa la misma rutina compartida.
func (u DeliverManual) runDelivery(ctx context.Context, dr deliveryRun) (DeliverResult, error) {
	return runDeliveryShared(ctx, runDeliveryDeps{
		Packages:   u.Packages,
		Deliveries: u.Deliveries,
		Outbox:     u.Outbox,
		TxRunner:   u.TxRunner,
	}, dr)
}

type runDeliveryDeps struct {
	Packages   domain.PackageRepository
	Deliveries domain.DeliveryRepository
	Outbox     domain.OutboxRepository
	TxRunner   TxRunner
}

func runDeliveryShared(ctx context.Context, deps runDeliveryDeps, dr deliveryRun) (DeliverResult, error) {
	var result DeliverResult
	run := func(txCtx context.Context) error {
		updated, err := deps.Packages.UpdateStatusOptimistic(
			txCtx,
			dr.Package.ID,
			dr.Package.Version,
			entities.PackageStatusDelivered,
			dr.ActorID,
		)
		if err != nil {
			return err
		}

		event, err := deps.Deliveries.Record(txCtx, dr.Input)
		if err != nil {
			return err
		}

		payload, perr := json.Marshal(map[string]any{
			"package_id":      updated.ID,
			"unit_id":         updated.UnitID,
			"delivered_at":    updated.DeliveredAt,
			"delivery_method": string(dr.Method),
			"event_id":        event.ID,
		})
		if perr != nil {
			return perr
		}
		if _, err := deps.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PackageID: updated.ID,
			EventType: entities.OutboxEventPackageDelivered,
			Payload:   payload,
		}); err != nil {
			return err
		}
		result = DeliverResult{Package: updated, Event: event}
		return nil
	}

	exec := func(c context.Context) error {
		if deps.TxRunner != nil {
			return deps.TxRunner.RunInTx(c, pgx.ReadCommitted, run)
		}
		return run(c)
	}

	if err := exec(ctx); err != nil {
		switch {
		case errors.Is(err, domain.ErrVersionConflict):
			return DeliverResult{}, mapVersionConflict()
		case errors.Is(err, domain.ErrPackageNotFound):
			return DeliverResult{}, apperrors.NotFound("package not found")
		default:
			return DeliverResult{}, apperrors.Internal("failed to deliver package")
		}
	}
	return result, nil
}

// hasNonEmpty devuelve true si el string pointer es no-nil y trimea a
// algo no vacio.
func hasNonEmpty(p *string) bool {
	if p == nil {
		return false
	}
	return strings.TrimSpace(*p) != ""
}

// mapInvalidTransition construye un Problem 409 con detalle del estado.
func mapInvalidTransition(current, next entities.PackageStatus) error {
	return apperrors.New(409, "invalid-transition", "Conflict",
		"cannot transition package from "+string(current)+" to "+string(next))
}

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"package was modified by another request; reload and retry")
}

// mapManualEvidenceRequired construye un Problem 400 estable para la
// regla "manual delivery requires signature or photo".
func mapManualEvidenceRequired() error {
	return apperrors.New(400, "evidence-required", "Bad Request",
		"signature_url or photo_evidence_url is required for manual delivery")
}
