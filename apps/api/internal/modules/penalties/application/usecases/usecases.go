// Package usecases orquesta la logica de aplicacion del modulo penalties.
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

	"github.com/saas-ph/api/internal/modules/penalties/domain"
	"github.com/saas-ph/api/internal/modules/penalties/domain/entities"
	"github.com/saas-ph/api/internal/modules/penalties/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// CreateCatalogEntry
// ---------------------------------------------------------------------------

// CreateCatalogEntry crea una entrada nueva en el catalogo de sanciones.
type CreateCatalogEntry struct {
	Catalog domain.CatalogRepository
}

// CreateCatalogEntryInput es el input del usecase.
type CreateCatalogEntryInput struct {
	Code                     string
	Name                     string
	Description              *string
	DefaultSanctionType      entities.SanctionType
	BaseAmount               float64
	RecurrenceMultiplier     float64
	RecurrenceCAPMultiplier  float64
	RequiresCouncilThreshold *float64
	ActorID                  string
}

// Execute valida y delega al repo.
func (u CreateCatalogEntry) Execute(ctx context.Context, in CreateCatalogEntryInput) (entities.PenaltyCatalog, error) {
	if strings.TrimSpace(in.Code) == "" {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("name is required")
	}
	if !in.DefaultSanctionType.IsValid() {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("default_sanction_type: invalid sanction type")
	}
	if in.BaseAmount < 0 {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("base_amount must be >= 0")
	}
	if in.RecurrenceMultiplier < 1 {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("recurrence_multiplier must be >= 1")
	}
	if in.RecurrenceCAPMultiplier < in.RecurrenceMultiplier {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("recurrence_cap_multiplier must be >= recurrence_multiplier")
	}

	entry, err := u.Catalog.Create(ctx, domain.CreateCatalogInput{
		Code:                     strings.TrimSpace(in.Code),
		Name:                     strings.TrimSpace(in.Name),
		Description:              in.Description,
		DefaultSanctionType:      in.DefaultSanctionType,
		BaseAmount:               in.BaseAmount,
		RecurrenceMultiplier:     in.RecurrenceMultiplier,
		RecurrenceCAPMultiplier:  in.RecurrenceCAPMultiplier,
		RequiresCouncilThreshold: in.RequiresCouncilThreshold,
		ActorID:                  in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCatalogCodeDuplicate) {
			return entities.PenaltyCatalog{}, apperrors.Conflict("catalog code already exists")
		}
		return entities.PenaltyCatalog{}, apperrors.Internal("failed to create catalog entry")
	}
	return entry, nil
}

// ---------------------------------------------------------------------------
// UpdateCatalogEntry
// ---------------------------------------------------------------------------

// UpdateCatalogEntry actualiza una entrada del catalogo con concurrencia
// optimista.
type UpdateCatalogEntry struct {
	Catalog domain.CatalogRepository
}

// UpdateCatalogEntryInput es el input del usecase.
type UpdateCatalogEntryInput struct {
	ID                       string
	Code                     string
	Name                     string
	Description              *string
	DefaultSanctionType      entities.SanctionType
	BaseAmount               float64
	RecurrenceMultiplier     float64
	RecurrenceCAPMultiplier  float64
	RequiresCouncilThreshold *float64
	Status                   entities.CatalogStatus
	ExpectedVersion          int32
	ActorID                  string
}

// Execute valida y delega al repo.
func (u UpdateCatalogEntry) Execute(ctx context.Context, in UpdateCatalogEntryInput) (entities.PenaltyCatalog, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("id: " + err.Error())
	}
	if strings.TrimSpace(in.Code) == "" {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("name is required")
	}
	if !in.DefaultSanctionType.IsValid() {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("default_sanction_type: invalid sanction type")
	}
	if !in.Status.IsValid() {
		return entities.PenaltyCatalog{}, apperrors.BadRequest("status: invalid catalog status")
	}

	entry, err := u.Catalog.Update(ctx, domain.UpdateCatalogInput{
		ID:                       in.ID,
		Code:                     strings.TrimSpace(in.Code),
		Name:                     strings.TrimSpace(in.Name),
		Description:              in.Description,
		DefaultSanctionType:      in.DefaultSanctionType,
		BaseAmount:               in.BaseAmount,
		RecurrenceMultiplier:     in.RecurrenceMultiplier,
		RecurrenceCAPMultiplier:  in.RecurrenceCAPMultiplier,
		RequiresCouncilThreshold: in.RequiresCouncilThreshold,
		Status:                   in.Status,
		ExpectedVersion:          in.ExpectedVersion,
		ActorID:                  in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCatalogNotFound):
			return entities.PenaltyCatalog{}, apperrors.NotFound("catalog entry not found")
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.PenaltyCatalog{}, mapVersionConflict()
		case errors.Is(err, domain.ErrCatalogCodeDuplicate):
			return entities.PenaltyCatalog{}, apperrors.Conflict("catalog code already exists")
		default:
			return entities.PenaltyCatalog{}, apperrors.Internal("failed to update catalog entry")
		}
	}
	return entry, nil
}

// ---------------------------------------------------------------------------
// ListCatalog
// ---------------------------------------------------------------------------

// ListCatalog lista las entradas activas del catalogo.
type ListCatalog struct {
	Catalog domain.CatalogRepository
}

// Execute delega al repo.
func (u ListCatalog) Execute(ctx context.Context) ([]entities.PenaltyCatalog, error) {
	out, err := u.Catalog.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list catalog")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// ImposePenalty
// ---------------------------------------------------------------------------

// ImposePenalty impone una sancion nueva en estado 'drafted', calculando
// el monto con reincidencia y evaluando si requiere aprobacion del consejo.
type ImposePenalty struct {
	Catalog   domain.CatalogRepository
	Penalties domain.PenaltyRepository
	History   domain.StatusHistoryRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// ImposePenaltyInput es el input del usecase.
type ImposePenaltyInput struct {
	CatalogID        string
	DebtorUserID     string
	UnitID           *string
	SourceIncidentID *string
	SanctionType     *entities.SanctionType
	Reason           string
	IdempotencyKey   *string
	ActorID          string
}

// Execute valida, calcula monto con reincidencia, y persiste.
func (u ImposePenalty) Execute(ctx context.Context, in ImposePenaltyInput) (entities.Penalty, error) {
	if err := policies.ValidateUUID(in.CatalogID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("catalog_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.DebtorUserID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("debtor_user_id: " + err.Error())
	}
	if in.UnitID != nil {
		if err := policies.ValidateUUID(*in.UnitID); err != nil {
			return entities.Penalty{}, apperrors.BadRequest("unit_id: " + err.Error())
		}
	}
	if in.SourceIncidentID != nil {
		if err := policies.ValidateUUID(*in.SourceIncidentID); err != nil {
			return entities.Penalty{}, apperrors.BadRequest("source_incident_id: " + err.Error())
		}
	}
	if strings.TrimSpace(in.Reason) == "" {
		return entities.Penalty{}, apperrors.BadRequest("reason is required")
	}

	// Fetch catalog entry.
	catalog, err := u.Catalog.GetByID(ctx, in.CatalogID)
	if err != nil {
		if errors.Is(err, domain.ErrCatalogNotFound) {
			return entities.Penalty{}, apperrors.NotFound("catalog entry not found")
		}
		return entities.Penalty{}, apperrors.Internal("failed to load catalog entry")
	}

	// Determine sanction type: use override or default from catalog.
	sanctionType := catalog.DefaultSanctionType
	if in.SanctionType != nil {
		if !in.SanctionType.IsValid() {
			return entities.Penalty{}, apperrors.BadRequest("sanction_type: invalid sanction type")
		}
		sanctionType = *in.SanctionType
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	// Calculate reincidence.
	since := now.AddDate(-1, 0, 0) // 365 days
	priorCount, err := u.Penalties.CountReincidence(ctx, in.DebtorUserID, in.CatalogID, since)
	if err != nil {
		return entities.Penalty{}, apperrors.Internal("failed to count reincidence")
	}

	amount := policies.CalculateReincidenceAmount(
		catalog.BaseAmount,
		catalog.RecurrenceMultiplier,
		catalog.RecurrenceCAPMultiplier,
		priorCount,
	)

	// Evaluate council approval.
	requiresCouncil := policies.RequiresCouncilApproval(amount, catalog.RequiresCouncilThreshold)

	var created entities.Penalty
	run := func(txCtx context.Context) error {
		penalty, createErr := u.Penalties.Create(txCtx, domain.CreatePenaltyInput{
			CatalogID:               in.CatalogID,
			DebtorUserID:            in.DebtorUserID,
			UnitID:                  in.UnitID,
			SourceIncidentID:        in.SourceIncidentID,
			SanctionType:            sanctionType,
			Amount:                  amount,
			Reason:                  strings.TrimSpace(in.Reason),
			ImposedByUserID:         in.ActorID,
			RequiresCouncilApproval: requiresCouncil,
			IdempotencyKey:          in.IdempotencyKey,
			ActorID:                 in.ActorID,
		})
		if createErr != nil {
			if errors.Is(createErr, domain.ErrPenaltyIdempotencyConflict) {
				return domain.ErrPenaltyIdempotencyConflict
			}
			return createErr
		}

		drafted := string(entities.PenaltyStatusDrafted)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			PenaltyID:            penalty.ID,
			FromStatus:           nil,
			ToStatus:             drafted,
			TransitionedByUserID: in.ActorID,
			Notes:                nil,
		}); hErr != nil {
			return hErr
		}

		created = penalty
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrPenaltyIdempotencyConflict) {
				return entities.Penalty{}, apperrors.Conflict("penalty with this idempotency key already exists")
			}
			return entities.Penalty{}, apperrors.Internal("failed to impose penalty")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrPenaltyIdempotencyConflict) {
				return entities.Penalty{}, apperrors.Conflict("penalty with this idempotency key already exists")
			}
			return entities.Penalty{}, apperrors.Internal("failed to impose penalty")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// NotifyPenalty
// ---------------------------------------------------------------------------

// NotifyPenalty transiciona una sancion de 'drafted' a 'notified'.
type NotifyPenalty struct {
	Penalties domain.PenaltyRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// NotifyPenaltyInput es el input del usecase.
type NotifyPenaltyInput struct {
	PenaltyID string
	ActorID   string
}

// Execute valida y ejecuta la transicion.
func (u NotifyPenalty) Execute(ctx context.Context, in NotifyPenaltyInput) (entities.Penalty, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.Penalty{}, apperrors.NotFound("penalty not found")
		}
		return entities.Penalty{}, apperrors.Internal("failed to load penalty")
	}

	if !policies.CanTransitionPenalty(penalty.Status, entities.PenaltyStatusNotified) {
		return entities.Penalty{}, apperrors.Conflict(
			"cannot notify penalty in status " + string(penalty.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	// Appeal deadline: 15 calendar days after notification.
	appealDeadline := now.AddDate(0, 0, 15)

	var updated entities.Penalty
	run := func(txCtx context.Context) error {
		result, setErr := u.Penalties.SetNotified(txCtx, penalty.ID, penalty.Version, now, appealDeadline, in.ActorID)
		if setErr != nil {
			return setErr
		}

		from := string(entities.PenaltyStatusDrafted)
		to := string(entities.PenaltyStatusNotified)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			PenaltyID:            result.ID,
			FromStatus:           &from,
			ToStatus:             to,
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"penalty_id":         result.ID,
			"debtor_user_id":     result.DebtorUserID,
			"sanction_type":      result.SanctionType,
			"amount":             result.Amount,
			"notified_at":        now,
			"appeal_deadline_at": appealDeadline,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PenaltyID: result.ID,
			EventType: entities.OutboxEventPenaltyNotified,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to notify penalty")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to notify penalty")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CouncilApprovePenalty
// ---------------------------------------------------------------------------

// CouncilApprovePenalty registra la aprobacion del consejo para una sancion
// que la requiere.
type CouncilApprovePenalty struct {
	Penalties domain.PenaltyRepository
	Now       func() time.Time
}

// CouncilApprovePenaltyInput es el input del usecase.
type CouncilApprovePenaltyInput struct {
	PenaltyID string
	ActorID   string
}

// Execute valida y delega.
func (u CouncilApprovePenalty) Execute(ctx context.Context, in CouncilApprovePenaltyInput) (entities.Penalty, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.Penalty{}, apperrors.NotFound("penalty not found")
		}
		return entities.Penalty{}, apperrors.Internal("failed to load penalty")
	}

	if !penalty.RequiresCouncilApproval {
		return entities.Penalty{}, apperrors.BadRequest("penalty does not require council approval")
	}
	if penalty.CouncilApprovedAt != nil {
		return entities.Penalty{}, apperrors.Conflict("penalty already has council approval")
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	result, err := u.Penalties.SetCouncilApproved(ctx, penalty.ID, penalty.Version, in.ActorID, now, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrVersionConflict) {
			return entities.Penalty{}, mapVersionConflict()
		}
		return entities.Penalty{}, apperrors.Internal("failed to set council approval")
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// ConfirmPenalty
// ---------------------------------------------------------------------------

// ConfirmPenalty transiciona una sancion de 'notified' a 'confirmed'.
// Si la sancion es monetaria, emite un evento penalty.charge_requested.
type ConfirmPenalty struct {
	Penalties domain.PenaltyRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// ConfirmPenaltyInput es el input del usecase.
type ConfirmPenaltyInput struct {
	PenaltyID string
	ActorID   string
}

// Execute valida y ejecuta la transicion.
func (u ConfirmPenalty) Execute(ctx context.Context, in ConfirmPenaltyInput) (entities.Penalty, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.Penalty{}, apperrors.NotFound("penalty not found")
		}
		return entities.Penalty{}, apperrors.Internal("failed to load penalty")
	}

	if !policies.CanTransitionPenalty(penalty.Status, entities.PenaltyStatusConfirmed) {
		return entities.Penalty{}, apperrors.Conflict(
			"cannot confirm penalty in status " + string(penalty.Status))
	}

	// If council approval is required, it must have been granted.
	if penalty.RequiresCouncilApproval && penalty.CouncilApprovedAt == nil {
		return entities.Penalty{}, apperrors.BadRequest("council approval is required before confirming")
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Penalty
	run := func(txCtx context.Context) error {
		result, setErr := u.Penalties.SetConfirmed(txCtx, penalty.ID, penalty.Version, now, in.ActorID)
		if setErr != nil {
			return setErr
		}

		from := string(penalty.Status)
		to := string(entities.PenaltyStatusConfirmed)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			PenaltyID:            result.ID,
			FromStatus:           &from,
			ToStatus:             to,
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		// Emit confirmed event.
		confirmedPayload, _ := json.Marshal(map[string]any{
			"penalty_id":     result.ID,
			"debtor_user_id": result.DebtorUserID,
			"sanction_type":  result.SanctionType,
			"amount":         result.Amount,
			"confirmed_at":   now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PenaltyID: result.ID,
			EventType: entities.OutboxEventPenaltyConfirmed,
			Payload:   confirmedPayload,
		}); oErr != nil {
			return oErr
		}

		// If monetary, emit charge_requested event.
		if result.SanctionType == entities.SanctionTypeMonetary {
			chargePayload, _ := json.Marshal(map[string]any{
				"penalty_id":     result.ID,
				"debtor_user_id": result.DebtorUserID,
				"unit_id":        result.UnitID,
				"amount":         result.Amount,
				"reason":         result.Reason,
				"confirmed_at":   now,
			})
			idemKey := "charge_" + result.ID
			if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
				PenaltyID:      result.ID,
				EventType:      entities.OutboxEventChargeRequested,
				Payload:        chargePayload,
				IdempotencyKey: &idemKey,
			}); oErr != nil {
				return oErr
			}
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to confirm penalty")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to confirm penalty")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// SettlePenalty
// ---------------------------------------------------------------------------

// SettlePenalty transiciona una sancion de 'confirmed' a 'settled'.
type SettlePenalty struct {
	Penalties domain.PenaltyRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// SettlePenaltyInput es el input del usecase.
type SettlePenaltyInput struct {
	PenaltyID string
	ActorID   string
}

// Execute valida y ejecuta la transicion.
func (u SettlePenalty) Execute(ctx context.Context, in SettlePenaltyInput) (entities.Penalty, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.Penalty{}, apperrors.NotFound("penalty not found")
		}
		return entities.Penalty{}, apperrors.Internal("failed to load penalty")
	}

	if !policies.CanTransitionPenalty(penalty.Status, entities.PenaltyStatusSettled) {
		return entities.Penalty{}, apperrors.Conflict(
			"cannot settle penalty in status " + string(penalty.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Penalty
	run := func(txCtx context.Context) error {
		result, setErr := u.Penalties.SetSettled(txCtx, penalty.ID, penalty.Version, now, in.ActorID)
		if setErr != nil {
			return setErr
		}

		from := string(entities.PenaltyStatusConfirmed)
		to := string(entities.PenaltyStatusSettled)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			PenaltyID:            result.ID,
			FromStatus:           &from,
			ToStatus:             to,
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"penalty_id":     result.ID,
			"debtor_user_id": result.DebtorUserID,
			"amount":         result.Amount,
			"settled_at":     now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PenaltyID: result.ID,
			EventType: entities.OutboxEventPenaltySettled,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to settle penalty")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to settle penalty")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CancelPenalty
// ---------------------------------------------------------------------------

// CancelPenalty transiciona una sancion de 'drafted' a 'cancelled'.
type CancelPenalty struct {
	Penalties domain.PenaltyRepository
	History   domain.StatusHistoryRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// CancelPenaltyInput es el input del usecase.
type CancelPenaltyInput struct {
	PenaltyID string
	ActorID   string
}

// Execute valida y ejecuta la transicion.
func (u CancelPenalty) Execute(ctx context.Context, in CancelPenaltyInput) (entities.Penalty, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.Penalty{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.Penalty{}, apperrors.NotFound("penalty not found")
		}
		return entities.Penalty{}, apperrors.Internal("failed to load penalty")
	}

	if !policies.CanTransitionPenalty(penalty.Status, entities.PenaltyStatusCancelled) {
		return entities.Penalty{}, apperrors.Conflict(
			"cannot cancel penalty in status " + string(penalty.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Penalty
	run := func(txCtx context.Context) error {
		result, setErr := u.Penalties.SetCancelled(txCtx, penalty.ID, penalty.Version, now, in.ActorID)
		if setErr != nil {
			return setErr
		}

		from := string(penalty.Status)
		to := string(entities.PenaltyStatusCancelled)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			PenaltyID:            result.ID,
			FromStatus:           &from,
			ToStatus:             to,
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to cancel penalty")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Penalty{}, mapVersionConflict()
			}
			return entities.Penalty{}, apperrors.Internal("failed to cancel penalty")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// SubmitAppeal
// ---------------------------------------------------------------------------

// SubmitAppeal crea una apelacion para un penalty en estado 'notified',
// transicionando el penalty a 'in_appeal'.
type SubmitAppeal struct {
	Penalties domain.PenaltyRepository
	Appeals   domain.AppealRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// SubmitAppealInput es el input del usecase.
type SubmitAppealInput struct {
	PenaltyID string
	Grounds   string
	ActorID   string
}

// Execute valida y ejecuta.
func (u SubmitAppeal) Execute(ctx context.Context, in SubmitAppealInput) (entities.PenaltyAppeal, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}
	if strings.TrimSpace(in.Grounds) == "" {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("grounds is required")
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.PenaltyAppeal{}, apperrors.NotFound("penalty not found")
		}
		return entities.PenaltyAppeal{}, apperrors.Internal("failed to load penalty")
	}

	if pErr := policies.CanAppeal(penalty.Status); pErr != nil {
		return entities.PenaltyAppeal{}, apperrors.Conflict(pErr.Error())
	}

	var created entities.PenaltyAppeal
	run := func(txCtx context.Context) error {
		appeal, createErr := u.Appeals.Create(txCtx, domain.CreateAppealInput{
			PenaltyID:         in.PenaltyID,
			SubmittedByUserID: in.ActorID,
			Grounds:           strings.TrimSpace(in.Grounds),
			ActorID:           in.ActorID,
		})
		if createErr != nil {
			if errors.Is(createErr, domain.ErrAppealAlreadyActive) {
				return domain.ErrAppealAlreadyActive
			}
			return createErr
		}

		// Transition penalty to in_appeal.
		if _, upErr := u.Penalties.UpdateStatus(txCtx, penalty.ID, penalty.Version, entities.PenaltyStatusInAppeal, in.ActorID); upErr != nil {
			return upErr
		}

		from := string(entities.PenaltyStatusNotified)
		to := string(entities.PenaltyStatusInAppeal)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			PenaltyID:            penalty.ID,
			FromStatus:           &from,
			ToStatus:             to,
			TransitionedByUserID: in.ActorID,
			Notes:                &appeal.Grounds,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"penalty_id":     penalty.ID,
			"appeal_id":      appeal.ID,
			"debtor_user_id": penalty.DebtorUserID,
			"grounds":        appeal.Grounds,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			PenaltyID: penalty.ID,
			EventType: entities.OutboxEventPenaltyAppealed,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		created = appeal
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrAppealAlreadyActive) {
				return entities.PenaltyAppeal{}, apperrors.Conflict("an active appeal already exists for this penalty")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.PenaltyAppeal{}, mapVersionConflict()
			}
			return entities.PenaltyAppeal{}, apperrors.Internal("failed to submit appeal")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrAppealAlreadyActive) {
				return entities.PenaltyAppeal{}, apperrors.Conflict("an active appeal already exists for this penalty")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.PenaltyAppeal{}, mapVersionConflict()
			}
			return entities.PenaltyAppeal{}, apperrors.Internal("failed to submit appeal")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// ResolveAppeal
// ---------------------------------------------------------------------------

// ResolveAppeal resuelve una apelacion existente. Si se acepta, el penalty
// pasa a 'dismissed'. Si se rechaza, pasa a 'confirmed'.
type ResolveAppeal struct {
	Penalties domain.PenaltyRepository
	Appeals   domain.AppealRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// ResolveAppealInput es el input del usecase.
type ResolveAppealInput struct {
	PenaltyID       string
	AppealID        string
	Resolution      string
	NewAppealStatus entities.AppealStatus
	ExpectedVersion int32
	ActorID         string
}

// Execute valida y ejecuta.
func (u ResolveAppeal) Execute(ctx context.Context, in ResolveAppealInput) (entities.PenaltyAppeal, error) {
	if err := policies.ValidateUUID(in.PenaltyID); err != nil {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("penalty_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.AppealID); err != nil {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("appeal_id: " + err.Error())
	}
	if strings.TrimSpace(in.Resolution) == "" {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("resolution is required")
	}
	if in.NewAppealStatus != entities.AppealStatusAccepted && in.NewAppealStatus != entities.AppealStatusRejected {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("status must be 'accepted' or 'rejected'")
	}

	appeal, err := u.Appeals.GetByID(ctx, in.AppealID)
	if err != nil {
		if errors.Is(err, domain.ErrAppealNotFound) {
			return entities.PenaltyAppeal{}, apperrors.NotFound("appeal not found")
		}
		return entities.PenaltyAppeal{}, apperrors.Internal("failed to load appeal")
	}
	if appeal.PenaltyID != in.PenaltyID {
		return entities.PenaltyAppeal{}, apperrors.BadRequest("appeal does not belong to this penalty")
	}

	if pErr := policies.CanResolveAppeal(appeal.Status); pErr != nil {
		return entities.PenaltyAppeal{}, apperrors.Conflict(pErr.Error())
	}

	penalty, err := u.Penalties.GetByID(ctx, in.PenaltyID)
	if err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return entities.PenaltyAppeal{}, apperrors.NotFound("penalty not found")
		}
		return entities.PenaltyAppeal{}, apperrors.Internal("failed to load penalty")
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var resolved entities.PenaltyAppeal
	run := func(txCtx context.Context) error {
		result, resolveErr := u.Appeals.Resolve(txCtx, domain.ResolveAppealInput{
			ID:               in.AppealID,
			ResolvedByUserID: in.ActorID,
			Resolution:       strings.TrimSpace(in.Resolution),
			NewStatus:        in.NewAppealStatus,
			ExpectedVersion:  in.ExpectedVersion,
			ActorID:          in.ActorID,
		})
		if resolveErr != nil {
			return resolveErr
		}

		// Transition penalty based on appeal outcome.
		if in.NewAppealStatus == entities.AppealStatusAccepted {
			// Accepted -> penalty dismissed.
			if _, dErr := u.Penalties.SetDismissed(txCtx, penalty.ID, penalty.Version, now, in.ActorID); dErr != nil {
				return dErr
			}

			from := string(entities.PenaltyStatusInAppeal)
			to := string(entities.PenaltyStatusDismissed)
			if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
				PenaltyID:            penalty.ID,
				FromStatus:           &from,
				ToStatus:             to,
				TransitionedByUserID: in.ActorID,
				Notes:                &in.Resolution,
			}); hErr != nil {
				return hErr
			}

			payload, _ := json.Marshal(map[string]any{
				"penalty_id":     penalty.ID,
				"appeal_id":      result.ID,
				"debtor_user_id": penalty.DebtorUserID,
				"dismissed_at":   now,
			})
			if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
				PenaltyID: penalty.ID,
				EventType: entities.OutboxEventPenaltyDismissed,
				Payload:   payload,
			}); oErr != nil {
				return oErr
			}
		} else {
			// Rejected -> penalty back to confirmed.
			if _, cErr := u.Penalties.SetConfirmed(txCtx, penalty.ID, penalty.Version, now, in.ActorID); cErr != nil {
				return cErr
			}

			from := string(entities.PenaltyStatusInAppeal)
			to := string(entities.PenaltyStatusConfirmed)
			if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
				PenaltyID:            penalty.ID,
				FromStatus:           &from,
				ToStatus:             to,
				TransitionedByUserID: in.ActorID,
				Notes:                &in.Resolution,
			}); hErr != nil {
				return hErr
			}

			confirmedPayload, _ := json.Marshal(map[string]any{
				"penalty_id":     penalty.ID,
				"debtor_user_id": penalty.DebtorUserID,
				"sanction_type":  penalty.SanctionType,
				"amount":         penalty.Amount,
				"confirmed_at":   now,
			})
			if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
				PenaltyID: penalty.ID,
				EventType: entities.OutboxEventPenaltyConfirmed,
				Payload:   confirmedPayload,
			}); oErr != nil {
				return oErr
			}

			// If monetary and rejected appeal -> emit charge_requested.
			if penalty.SanctionType == entities.SanctionTypeMonetary {
				chargePayload, _ := json.Marshal(map[string]any{
					"penalty_id":     penalty.ID,
					"debtor_user_id": penalty.DebtorUserID,
					"unit_id":        penalty.UnitID,
					"amount":         penalty.Amount,
					"reason":         penalty.Reason,
					"confirmed_at":   now,
				})
				idemKey := "charge_" + penalty.ID
				if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
					PenaltyID:      penalty.ID,
					EventType:      entities.OutboxEventChargeRequested,
					Payload:        chargePayload,
					IdempotencyKey: &idemKey,
				}); oErr != nil {
					return oErr
				}
			}
		}

		resolved = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.PenaltyAppeal{}, mapVersionConflict()
			}
			return entities.PenaltyAppeal{}, apperrors.Internal("failed to resolve appeal")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.PenaltyAppeal{}, mapVersionConflict()
			}
			return entities.PenaltyAppeal{}, apperrors.Internal("failed to resolve appeal")
		}
	}
	return resolved, nil
}

// ---------------------------------------------------------------------------
// ListPenalties
// ---------------------------------------------------------------------------

// ListPenalties lista las sanciones.
type ListPenalties struct {
	Penalties domain.PenaltyRepository
}

// Execute delega al repo.
func (u ListPenalties) Execute(ctx context.Context) ([]entities.Penalty, error) {
	out, err := u.Penalties.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list penalties")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// GetPenaltyHistory
// ---------------------------------------------------------------------------

// GetPenaltyHistory devuelve el historial de transiciones de una sancion.
type GetPenaltyHistory struct {
	Penalties domain.PenaltyRepository
	History   domain.StatusHistoryRepository
}

// Execute valida y delega.
func (u GetPenaltyHistory) Execute(ctx context.Context, penaltyID string) ([]entities.PenaltyStatusHistory, error) {
	if err := policies.ValidateUUID(penaltyID); err != nil {
		return nil, apperrors.BadRequest("penalty_id: " + err.Error())
	}

	// Validate penalty exists.
	if _, err := u.Penalties.GetByID(ctx, penaltyID); err != nil {
		if errors.Is(err, domain.ErrPenaltyNotFound) {
			return nil, apperrors.NotFound("penalty not found")
		}
		return nil, apperrors.Internal("failed to load penalty")
	}

	out, err := u.History.ListByPenaltyID(ctx, penaltyID)
	if err != nil {
		return nil, apperrors.Internal("failed to list penalty history")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}
