// Package persistence implementa los puertos del modulo penalties usando
// el codigo generado por sqlc.
//
// Reglas:
//   - El pool del Tenant DB se obtiene del contexto via tenantctx.FromCtx.
//   - NO se usa database/sql ni SQL inline.
//   - Las usecases que requieren atomicidad multi-tabla pasan un pgx.Tx
//     en el contexto via WithTx(ctx, tx). Si esta presente, los repos lo
//     usan; si no, usan el pool del tenant.
package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/penalties/domain"
	"github.com/saas-ph/api/internal/modules/penalties/domain/entities"
	penaltiesdb "github.com/saas-ph/api/internal/modules/penalties/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// --- ctx helper para transaccion ---

type txCtxKey struct{}

// WithTx inyecta una transaccion pgx en el contexto. Cuando los repos
// resuelvan su Querier, prefieren la tx si esta presente.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// txFromCtx extrae una tx pgx del contexto si existe.
func txFromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	return tx, ok
}

func querier(ctx context.Context) (*penaltiesdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return penaltiesdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("penalties: tenant pool is nil")
	}
	return penaltiesdb.New(t.Pool), nil
}

// --- CatalogRepository ---

// CatalogRepository implementa domain.CatalogRepository.
type CatalogRepository struct{}

// NewCatalogRepository construye una instancia stateless.
func NewCatalogRepository() *CatalogRepository { return &CatalogRepository{} }

// Create implementa domain.CatalogRepository.
func (r *CatalogRepository) Create(ctx context.Context, in domain.CreateCatalogInput) (entities.PenaltyCatalog, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyCatalog{}, err
	}
	row, err := q.CreatePenaltyCatalogEntry(ctx, penaltiesdb.CreatePenaltyCatalogEntryParams{
		Code:                     in.Code,
		Name:                     in.Name,
		Description:              in.Description,
		DefaultSanctionType:      string(in.DefaultSanctionType),
		BaseAmount:               float64ToNumeric(in.BaseAmount),
		RecurrenceMultiplier:     float64ToNumeric(in.RecurrenceMultiplier),
		RecurrenceCapMultiplier:  float64ToNumeric(in.RecurrenceCAPMultiplier),
		RequiresCouncilThreshold: float64PtrToNumeric(in.RequiresCouncilThreshold),
		CreatedBy:                uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.PenaltyCatalog{}, domain.ErrCatalogCodeDuplicate
		}
		return entities.PenaltyCatalog{}, err
	}
	return mapCatalog(row), nil
}

// GetByID implementa domain.CatalogRepository.
func (r *CatalogRepository) GetByID(ctx context.Context, id string) (entities.PenaltyCatalog, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyCatalog{}, err
	}
	row, err := q.GetPenaltyCatalogByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PenaltyCatalog{}, domain.ErrCatalogNotFound
		}
		return entities.PenaltyCatalog{}, err
	}
	return mapCatalog(row), nil
}

// List implementa domain.CatalogRepository.
func (r *CatalogRepository) List(ctx context.Context) ([]entities.PenaltyCatalog, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPenaltyCatalog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.PenaltyCatalog, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCatalog(row))
	}
	return out, nil
}

// Update implementa domain.CatalogRepository.
func (r *CatalogRepository) Update(ctx context.Context, in domain.UpdateCatalogInput) (entities.PenaltyCatalog, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyCatalog{}, err
	}
	row, err := q.UpdatePenaltyCatalogEntry(ctx, penaltiesdb.UpdatePenaltyCatalogEntryParams{
		NewCode:                     in.Code,
		NewName:                     in.Name,
		NewDescription:              in.Description,
		NewDefaultSanctionType:      string(in.DefaultSanctionType),
		NewBaseAmount:               float64ToNumeric(in.BaseAmount),
		NewRecurrenceMultiplier:     float64ToNumeric(in.RecurrenceMultiplier),
		NewRecurrenceCapMultiplier:  float64ToNumeric(in.RecurrenceCAPMultiplier),
		NewRequiresCouncilThreshold: float64PtrToNumeric(in.RequiresCouncilThreshold),
		NewStatus:                   string(in.Status),
		UpdatedBy:                   uuidToPgtype(in.ActorID),
		ID:                          uuidToPgtype(in.ID),
		ExpectedVersion:             in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PenaltyCatalog{}, domain.ErrVersionConflict
		}
		if isUniqueViolation(err) {
			return entities.PenaltyCatalog{}, domain.ErrCatalogCodeDuplicate
		}
		return entities.PenaltyCatalog{}, err
	}
	return mapCatalog(row), nil
}

// --- PenaltyRepository ---

// PenaltyRepository implementa domain.PenaltyRepository.
type PenaltyRepository struct{}

// NewPenaltyRepository construye una instancia stateless.
func NewPenaltyRepository() *PenaltyRepository { return &PenaltyRepository{} }

// Create implementa domain.PenaltyRepository.
func (r *PenaltyRepository) Create(ctx context.Context, in domain.CreatePenaltyInput) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.CreatePenalty(ctx, penaltiesdb.CreatePenaltyParams{
		CatalogID:               uuidToPgtype(in.CatalogID),
		DebtorUserID:            uuidToPgtype(in.DebtorUserID),
		UnitID:                  uuidToPgtypePtr(in.UnitID),
		SourceIncidentID:        uuidToPgtypePtr(in.SourceIncidentID),
		SanctionType:            string(in.SanctionType),
		Amount:                  float64ToNumeric(in.Amount),
		Reason:                  in.Reason,
		ImposedByUserID:         uuidToPgtype(in.ImposedByUserID),
		RequiresCouncilApproval: in.RequiresCouncilApproval,
		IdempotencyKey:          in.IdempotencyKey,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Penalty{}, domain.ErrPenaltyIdempotencyConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// GetByID implementa domain.PenaltyRepository.
func (r *PenaltyRepository) GetByID(ctx context.Context, id string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.GetPenaltyByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrPenaltyNotFound
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// List implementa domain.PenaltyRepository.
func (r *PenaltyRepository) List(ctx context.Context) ([]entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPenalties(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Penalty, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPenalty(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.PenaltyRepository.
func (r *PenaltyRepository) UpdateStatus(ctx context.Context, id string, expectedVersion int32, newStatus entities.PenaltyStatus, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.UpdatePenaltyStatus(ctx, penaltiesdb.UpdatePenaltyStatusParams{
		NewStatus:       string(newStatus),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// SetNotified implementa domain.PenaltyRepository.
func (r *PenaltyRepository) SetNotified(ctx context.Context, id string, expectedVersion int32, notifiedAt time.Time, appealDeadlineAt time.Time, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.SetPenaltyNotified(ctx, penaltiesdb.SetPenaltyNotifiedParams{
		NotifiedAt:       timeToPgTimestamptz(notifiedAt),
		AppealDeadlineAt: timeToPgTimestamptz(appealDeadlineAt),
		UpdatedBy:        uuidToPgtype(actorID),
		ID:               uuidToPgtype(id),
		ExpectedVersion:  expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// SetCouncilApproved implementa domain.PenaltyRepository.
func (r *PenaltyRepository) SetCouncilApproved(ctx context.Context, id string, expectedVersion int32, approvedByUserID string, approvedAt time.Time, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.SetPenaltyCouncilApproved(ctx, penaltiesdb.SetPenaltyCouncilApprovedParams{
		CouncilApprovedBy: uuidToPgtype(approvedByUserID),
		CouncilApprovedAt: timeToPgTimestamptz(approvedAt),
		UpdatedBy:         uuidToPgtype(actorID),
		ID:                uuidToPgtype(id),
		ExpectedVersion:   expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// SetConfirmed implementa domain.PenaltyRepository.
func (r *PenaltyRepository) SetConfirmed(ctx context.Context, id string, expectedVersion int32, confirmedAt time.Time, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.SetPenaltyConfirmed(ctx, penaltiesdb.SetPenaltyConfirmedParams{
		ConfirmedAt:     timeToPgTimestamptz(confirmedAt),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// SetSettled implementa domain.PenaltyRepository.
func (r *PenaltyRepository) SetSettled(ctx context.Context, id string, expectedVersion int32, settledAt time.Time, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.SetPenaltySettled(ctx, penaltiesdb.SetPenaltySettledParams{
		SettledAt:       timeToPgTimestamptz(settledAt),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// SetDismissed implementa domain.PenaltyRepository.
func (r *PenaltyRepository) SetDismissed(ctx context.Context, id string, expectedVersion int32, dismissedAt time.Time, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.SetPenaltyDismissed(ctx, penaltiesdb.SetPenaltyDismissedParams{
		DismissedAt:     timeToPgTimestamptz(dismissedAt),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// SetCancelled implementa domain.PenaltyRepository.
func (r *PenaltyRepository) SetCancelled(ctx context.Context, id string, expectedVersion int32, cancelledAt time.Time, actorID string) (entities.Penalty, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Penalty{}, err
	}
	row, err := q.SetPenaltyCancelled(ctx, penaltiesdb.SetPenaltyCancelledParams{
		CancelledAt:     timeToPgTimestamptz(cancelledAt),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Penalty{}, domain.ErrVersionConflict
		}
		return entities.Penalty{}, err
	}
	return mapPenalty(row), nil
}

// CountReincidence implementa domain.PenaltyRepository.
func (r *PenaltyRepository) CountReincidence(ctx context.Context, debtorUserID, catalogID string, since time.Time) (int, error) {
	q, err := querier(ctx)
	if err != nil {
		return 0, err
	}
	cnt, err := q.CountPenaltyReincidence(ctx,
		uuidToPgtype(debtorUserID),
		uuidToPgtype(catalogID),
		timeToPgTimestamptz(since),
	)
	if err != nil {
		return 0, err
	}
	return int(cnt), nil
}

// --- AppealRepository ---

// AppealRepository implementa domain.AppealRepository.
type AppealRepository struct{}

// NewAppealRepository construye una instancia stateless.
func NewAppealRepository() *AppealRepository { return &AppealRepository{} }

// Create implementa domain.AppealRepository.
func (r *AppealRepository) Create(ctx context.Context, in domain.CreateAppealInput) (entities.PenaltyAppeal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyAppeal{}, err
	}
	row, err := q.CreatePenaltyAppeal(ctx, penaltiesdb.CreatePenaltyAppealParams{
		PenaltyID:         uuidToPgtype(in.PenaltyID),
		SubmittedByUserID: uuidToPgtype(in.SubmittedByUserID),
		Grounds:           in.Grounds,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.PenaltyAppeal{}, domain.ErrAppealAlreadyActive
		}
		return entities.PenaltyAppeal{}, err
	}
	return mapAppeal(row), nil
}

// GetByID implementa domain.AppealRepository.
func (r *AppealRepository) GetByID(ctx context.Context, id string) (entities.PenaltyAppeal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyAppeal{}, err
	}
	row, err := q.GetPenaltyAppealByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PenaltyAppeal{}, domain.ErrAppealNotFound
		}
		return entities.PenaltyAppeal{}, err
	}
	return mapAppeal(row), nil
}

// GetActiveByPenaltyID implementa domain.AppealRepository.
func (r *AppealRepository) GetActiveByPenaltyID(ctx context.Context, penaltyID string) (entities.PenaltyAppeal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyAppeal{}, err
	}
	row, err := q.GetActiveAppealByPenaltyID(ctx, uuidToPgtype(penaltyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PenaltyAppeal{}, domain.ErrAppealNotFound
		}
		return entities.PenaltyAppeal{}, err
	}
	return mapAppeal(row), nil
}

// Resolve implementa domain.AppealRepository.
func (r *AppealRepository) Resolve(ctx context.Context, in domain.ResolveAppealInput) (entities.PenaltyAppeal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyAppeal{}, err
	}
	row, err := q.ResolvePenaltyAppeal(ctx, penaltiesdb.ResolvePenaltyAppealParams{
		ResolvedBy:      uuidToPgtype(in.ResolvedByUserID),
		Resolution:      in.Resolution,
		NewStatus:       string(in.NewStatus),
		UpdatedBy:       uuidToPgtype(in.ActorID),
		ID:              uuidToPgtype(in.ID),
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PenaltyAppeal{}, domain.ErrVersionConflict
		}
		return entities.PenaltyAppeal{}, err
	}
	return mapAppeal(row), nil
}

// --- StatusHistoryRepository ---

// StatusHistoryRepository implementa domain.StatusHistoryRepository.
type StatusHistoryRepository struct{}

// NewStatusHistoryRepository construye una instancia stateless.
func NewStatusHistoryRepository() *StatusHistoryRepository { return &StatusHistoryRepository{} }

// Record implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) Record(ctx context.Context, in domain.RecordStatusHistoryInput) (entities.PenaltyStatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PenaltyStatusHistory{}, err
	}
	row, err := q.RecordPenaltyStatusHistory(ctx, penaltiesdb.RecordPenaltyStatusHistoryParams{
		PenaltyID:            uuidToPgtype(in.PenaltyID),
		FromStatus:           in.FromStatus,
		ToStatus:             in.ToStatus,
		TransitionedByUserID: uuidToPgtype(in.TransitionedByUserID),
		Notes:                in.Notes,
	})
	if err != nil {
		return entities.PenaltyStatusHistory{}, err
	}
	return mapStatusHistory(row), nil
}

// ListByPenaltyID implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) ListByPenaltyID(ctx context.Context, penaltyID string) ([]entities.PenaltyStatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPenaltyStatusHistory(ctx, uuidToPgtype(penaltyID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.PenaltyStatusHistory, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapStatusHistory(row))
	}
	return out, nil
}

// --- OutboxRepository ---

// OutboxRepository implementa domain.OutboxRepository.
type OutboxRepository struct{}

// NewOutboxRepository construye una instancia stateless.
func NewOutboxRepository() *OutboxRepository { return &OutboxRepository{} }

// Enqueue implementa domain.OutboxRepository.
func (r *OutboxRepository) Enqueue(ctx context.Context, in domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	row, err := q.EnqueuePenaltyOutboxEvent(ctx, penaltiesdb.EnqueuePenaltyOutboxEventParams{
		PenaltyID:      uuidToPgtype(in.PenaltyID),
		EventType:      string(in.EventType),
		Payload:        in.Payload,
		IdempotencyKey: in.IdempotencyKey,
	})
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	return mapOutbox(row), nil
}

// LockPending implementa domain.OutboxRepository.
func (r *OutboxRepository) LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.LockPendingPenaltyOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]entities.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOutbox(row))
	}
	return out, nil
}

// MarkDelivered implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkDelivered(ctx context.Context, id string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.MarkPenaltyOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkPenaltyOutboxEventFailed(ctx, penaltiesdb.MarkPenaltyOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// --- helpers de mapeo ---

func mapCatalog(r penaltiesdb.PenaltyCatalog) entities.PenaltyCatalog {
	out := entities.PenaltyCatalog{
		ID:                  uuidString(r.ID),
		Code:                r.Code,
		Name:                r.Name,
		Description:         r.Description,
		DefaultSanctionType: entities.SanctionType(r.DefaultSanctionType),
		Status:              entities.CatalogStatus(r.Status),
		CreatedAt:           tsToTime(r.CreatedAt),
		UpdatedAt:           tsToTime(r.UpdatedAt),
		Version:             r.Version,
	}
	if f, err := numericToFloat64(r.BaseAmount); err == nil {
		out.BaseAmount = f
	}
	if f, err := numericToFloat64(r.RecurrenceMultiplier); err == nil {
		out.RecurrenceMultiplier = f
	}
	if f, err := numericToFloat64(r.RecurrenceCapMultiplier); err == nil {
		out.RecurrenceCAPMultiplier = f
	}
	if r.RequiresCouncilThreshold.Valid {
		if f, err := numericToFloat64(r.RequiresCouncilThreshold); err == nil {
			out.RequiresCouncilThreshold = &f
		}
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapPenalty(r penaltiesdb.Penalty) entities.Penalty {
	out := entities.Penalty{
		ID:                      uuidString(r.ID),
		CatalogID:               uuidString(r.CatalogID),
		DebtorUserID:            uuidString(r.DebtorUserID),
		SanctionType:            entities.SanctionType(r.SanctionType),
		Reason:                  r.Reason,
		ImposedByUserID:         uuidString(r.ImposedByUserID),
		RequiresCouncilApproval: r.RequiresCouncilApproval,
		IdempotencyKey:          r.IdempotencyKey,
		Status:                  entities.PenaltyStatus(r.Status),
		CreatedAt:               tsToTime(r.CreatedAt),
		UpdatedAt:               tsToTime(r.UpdatedAt),
		Version:                 r.Version,
	}
	if f, err := numericToFloat64(r.Amount); err == nil {
		out.Amount = f
	}
	if s := uuidStringPtr(r.UnitID); s != nil {
		out.UnitID = s
	}
	if s := uuidStringPtr(r.SourceIncidentID); s != nil {
		out.SourceIncidentID = s
	}
	if r.NotifiedAt.Valid {
		t := r.NotifiedAt.Time
		out.NotifiedAt = &t
	}
	if r.AppealDeadlineAt.Valid {
		t := r.AppealDeadlineAt.Time
		out.AppealDeadlineAt = &t
	}
	if r.ConfirmedAt.Valid {
		t := r.ConfirmedAt.Time
		out.ConfirmedAt = &t
	}
	if r.SettledAt.Valid {
		t := r.SettledAt.Time
		out.SettledAt = &t
	}
	if r.DismissedAt.Valid {
		t := r.DismissedAt.Time
		out.DismissedAt = &t
	}
	if r.CancelledAt.Valid {
		t := r.CancelledAt.Time
		out.CancelledAt = &t
	}
	if s := uuidStringPtr(r.CouncilApprovedByUserID); s != nil {
		out.CouncilApprovedByUserID = s
	}
	if r.CouncilApprovedAt.Valid {
		t := r.CouncilApprovedAt.Time
		out.CouncilApprovedAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapAppeal(r penaltiesdb.PenaltyAppeal) entities.PenaltyAppeal {
	out := entities.PenaltyAppeal{
		ID:                uuidString(r.ID),
		PenaltyID:         uuidString(r.PenaltyID),
		SubmittedByUserID: uuidString(r.SubmittedByUserID),
		SubmittedAt:       tsToTime(r.SubmittedAt),
		Grounds:           r.Grounds,
		Resolution:        r.Resolution,
		Status:            entities.AppealStatus(r.Status),
		CreatedAt:         tsToTime(r.CreatedAt),
		UpdatedAt:         tsToTime(r.UpdatedAt),
		Version:           r.Version,
	}
	if s := uuidStringPtr(r.ResolvedByUserID); s != nil {
		out.ResolvedByUserID = s
	}
	if r.ResolvedAt.Valid {
		t := r.ResolvedAt.Time
		out.ResolvedAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapStatusHistory(r penaltiesdb.PenaltyStatusHistory) entities.PenaltyStatusHistory {
	out := entities.PenaltyStatusHistory{
		ID:                   uuidString(r.ID),
		PenaltyID:            uuidString(r.PenaltyID),
		FromStatus:           r.FromStatus,
		ToStatus:             r.ToStatus,
		TransitionedByUserID: uuidString(r.TransitionedByUserID),
		TransitionedAt:       tsToTime(r.TransitionedAt),
		Notes:                r.Notes,
		Status:               r.Status,
		CreatedAt:            tsToTime(r.CreatedAt),
		UpdatedAt:            tsToTime(r.UpdatedAt),
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	return out
}

func mapOutbox(r penaltiesdb.PenaltyOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:             uuidString(r.ID),
		PenaltyID:      uuidString(r.PenaltyID),
		EventType:      entities.OutboxEventType(r.EventType),
		Payload:        r.Payload,
		IdempotencyKey: r.IdempotencyKey,
		CreatedAt:      tsToTime(r.CreatedAt),
		NextAttemptAt:  tsToTime(r.NextAttemptAt),
		Attempts:       r.Attempts,
		LastError:      r.LastError,
	}
	if r.DeliveredAt.Valid {
		t := r.DeliveredAt.Time
		out.DeliveredAt = &t
	}
	return out
}

// --- pgtype helpers ---

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	v, err := u.Value()
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	if s == "" {
		return nil
	}
	return &s
}

func uuidToPgtype(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

func uuidToPgtypePtr(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{Valid: false}
	}
	return uuidToPgtype(*s)
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(f); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
}

func float64PtrToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	return float64ToNumeric(*f)
}

func numericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, errors.New("numeric is null")
	}
	v, err := n.Value()
	if err != nil {
		return 0, err
	}
	switch val := v.(type) {
	case float64:
		return val, nil
	case string:
		var num pgtype.Numeric
		if sErr := num.Scan(val); sErr != nil {
			return 0, sErr
		}
		f64, f64Err := num.Float64Value()
		if f64Err != nil {
			return 0, f64Err
		}
		return f64.Float64, nil
	default:
		return 0, errors.New("unexpected numeric type")
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

// Compile-time checks: each repo implements the domain port.
var (
	_ domain.CatalogRepository       = (*CatalogRepository)(nil)
	_ domain.PenaltyRepository       = (*PenaltyRepository)(nil)
	_ domain.AppealRepository        = (*AppealRepository)(nil)
	_ domain.StatusHistoryRepository = (*StatusHistoryRepository)(nil)
	_ domain.OutboxRepository        = (*OutboxRepository)(nil)
)
