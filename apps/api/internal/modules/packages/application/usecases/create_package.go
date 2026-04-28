// Package usecases orquesta la logica de aplicacion del modulo packages.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	"github.com/saas-ph/api/internal/modules/packages/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// CreatePackage crea un paquete nuevo en estado 'received' y encola un
// outbox event 'package.received' EN LA MISMA transaccion (ADR 0005).
//
// Reglas:
//   - UnitID, RecipientName y ReceivedByUserID son obligatorios.
//   - Si CategoryID o CategoryName vienen, se resuelve la categoria; si
//     la categoria requiere evidencia y ReceivedEvidenceURL no esta
//     presente => 400 ErrEvidenceRequired.
//   - INSERT paquete + INSERT outbox van EN UNA SOLA transaccion (TxRunner).
type CreatePackage struct {
	Packages   domain.PackageRepository
	Categories domain.CategoryRepository
	Outbox     domain.OutboxRepository
	TxRunner   TxRunner
	// Now permite inyectar reloj para tests; si nil, time.Now.
	Now func() time.Time
}

// CreatePackageInput es el input del usecase (sin tags JSON).
type CreatePackageInput struct {
	UnitID              string
	RecipientName       string
	CategoryID          *string
	CategoryName        *string
	ReceivedEvidenceURL *string
	Carrier             *string
	TrackingNumber      *string
	ReceivedByUserID    string
}

// Execute valida y delega al repo.
func (u CreatePackage) Execute(ctx context.Context, in CreatePackageInput) (entities.Package, error) {
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.Package{}, apperrors.BadRequest("unit_id: " + err.Error())
	}
	if err := policies.ValidateRecipientName(strings.TrimSpace(in.RecipientName)); err != nil {
		return entities.Package{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateUUID(in.ReceivedByUserID); err != nil {
		return entities.Package{}, apperrors.BadRequest("received_by_user_id: " + err.Error())
	}

	// Resuelve categoria si vino. CategoryID prevalece sobre CategoryName.
	var category *entities.PackageCategory
	if in.CategoryID != nil && *in.CategoryID != "" {
		if err := policies.ValidateUUID(*in.CategoryID); err != nil {
			return entities.Package{}, apperrors.BadRequest("category_id: " + err.Error())
		}
		c, err := u.Categories.GetByID(ctx, *in.CategoryID)
		if err != nil {
			if errors.Is(err, domain.ErrCategoryNotFound) {
				return entities.Package{}, apperrors.NotFound("category not found")
			}
			return entities.Package{}, apperrors.Internal("failed to load category")
		}
		category = &c
	} else if in.CategoryName != nil && strings.TrimSpace(*in.CategoryName) != "" {
		c, err := u.Categories.GetByName(ctx, strings.TrimSpace(*in.CategoryName))
		if err != nil {
			if errors.Is(err, domain.ErrCategoryNotFound) {
				return entities.Package{}, apperrors.NotFound("category not found")
			}
			return entities.Package{}, apperrors.Internal("failed to load category")
		}
		category = &c
	}

	// Si la categoria requiere evidencia, se exige URL.
	if policies.RequiresEvidence(category) {
		if in.ReceivedEvidenceURL == nil || strings.TrimSpace(*in.ReceivedEvidenceURL) == "" {
			return entities.Package{}, mapEvidenceRequired()
		}
	}

	var categoryID *string
	if category != nil {
		id := category.ID
		categoryID = &id
	}

	createIn := domain.CreatePackageInput{
		UnitID:              in.UnitID,
		RecipientName:       strings.TrimSpace(in.RecipientName),
		CategoryID:          categoryID,
		ReceivedEvidenceURL: in.ReceivedEvidenceURL,
		Carrier:             in.Carrier,
		TrackingNumber:      in.TrackingNumber,
		ReceivedByUserID:    in.ReceivedByUserID,
	}

	var created entities.Package
	run := func(txCtx context.Context) error {
		pkg, err := u.Packages.Create(txCtx, createIn)
		if err != nil {
			return err
		}
		payload, perr := json.Marshal(map[string]any{
			"package_id":     pkg.ID,
			"unit_id":        pkg.UnitID,
			"recipient_name": pkg.RecipientName,
			"received_at":    pkg.ReceivedAt,
		})
		if perr != nil {
			return perr
		}
		if _, err := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PackageID: pkg.ID,
			EventType: entities.OutboxEventPackageReceived,
			Payload:   payload,
		}); err != nil {
			return err
		}
		created = pkg
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			return entities.Package{}, apperrors.Internal("failed to create package")
		}
	} else {
		// Sin TxRunner los repos operan sobre el pool directamente. Util
		// para tests con mocks que no requieren tx.
		if err := run(ctx); err != nil {
			return entities.Package{}, apperrors.Internal("failed to create package")
		}
	}
	return created, nil
}

// mapEvidenceRequired construye un Problem 400 con slug estable.
func mapEvidenceRequired() error {
	return apperrors.New(400, "evidence-required", "Bad Request",
		"received_evidence_url is required for this category")
}
