// Package persistence implementa los puertos del modulo reservations usando
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

	"github.com/saas-ph/api/internal/modules/reservations/domain"
	"github.com/saas-ph/api/internal/modules/reservations/domain/entities"
	resdb "github.com/saas-ph/api/internal/modules/reservations/infrastructure/persistence/sqlcgen"
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

func querier(ctx context.Context) (*resdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return resdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("reservations: tenant pool is nil")
	}
	return resdb.New(t.Pool), nil
}

// --- CommonAreaRepository ---

// CommonAreaRepository implementa domain.CommonAreaRepository.
type CommonAreaRepository struct{}

// NewCommonAreaRepository construye una instancia stateless.
func NewCommonAreaRepository() *CommonAreaRepository { return &CommonAreaRepository{} }

// Create implementa domain.CommonAreaRepository.
func (r *CommonAreaRepository) Create(ctx context.Context, in domain.CreateCommonAreaInput) (entities.CommonArea, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.CommonArea{}, err
	}
	row, err := q.CreateCommonArea(ctx, resdb.CreateCommonAreaParams{
		Code:                in.Code,
		Name:                in.Name,
		Kind:                string(in.Kind),
		MaxCapacity:         in.MaxCapacity,
		OpeningTime:         in.OpeningTime,
		ClosingTime:         in.ClosingTime,
		SlotDurationMinutes: in.SlotDurationMinutes,
		CostPerUse:          float64ToNumeric(in.CostPerUse),
		SecurityDeposit:     float64ToNumeric(in.SecurityDeposit),
		RequiresApproval:    in.RequiresApproval,
		IsActive:            in.IsActive,
		Description:         in.Description,
		CreatedBy:           pgtype.UUID{Valid: false},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.CommonArea{}, domain.ErrCommonAreaCodeDuplicate
		}
		return entities.CommonArea{}, err
	}
	return mapCommonArea(row), nil
}

// GetByID implementa domain.CommonAreaRepository.
func (r *CommonAreaRepository) GetByID(ctx context.Context, id string) (entities.CommonArea, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.CommonArea{}, err
	}
	row, err := q.GetCommonAreaByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.CommonArea{}, domain.ErrCommonAreaNotFound
		}
		return entities.CommonArea{}, err
	}
	return mapCommonArea(row), nil
}

// List implementa domain.CommonAreaRepository.
func (r *CommonAreaRepository) List(ctx context.Context) ([]entities.CommonArea, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListCommonAreas(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.CommonArea, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCommonArea(row))
	}
	return out, nil
}

// Update implementa domain.CommonAreaRepository.
func (r *CommonAreaRepository) Update(ctx context.Context, in domain.UpdateCommonAreaInput) (entities.CommonArea, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.CommonArea{}, err
	}
	row, err := q.UpdateCommonArea(ctx, resdb.UpdateCommonAreaParams{
		NewCode:                in.Code,
		NewName:                in.Name,
		NewKind:                string(in.Kind),
		NewMaxCapacity:         in.MaxCapacity,
		NewOpeningTime:         in.OpeningTime,
		NewClosingTime:         in.ClosingTime,
		NewSlotDurationMinutes: in.SlotDurationMinutes,
		NewCostPerUse:          float64ToNumeric(in.CostPerUse),
		NewSecurityDeposit:     float64ToNumeric(in.SecurityDeposit),
		NewRequiresApproval:    in.RequiresApproval,
		NewIsActive:            in.IsActive,
		NewDescription:         in.Description,
		NewStatus:              string(in.Status),
		UpdatedBy:              uuidToPgtype(in.ActorID),
		ID:                     uuidToPgtype(in.ID),
		ExpectedVersion:        in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.CommonArea{}, domain.ErrVersionConflict
		}
		if isUniqueViolation(err) {
			return entities.CommonArea{}, domain.ErrCommonAreaCodeDuplicate
		}
		return entities.CommonArea{}, err
	}
	return mapCommonArea(row), nil
}

// --- BlackoutRepository ---

// BlackoutRepository implementa domain.BlackoutRepository.
type BlackoutRepository struct{}

// NewBlackoutRepository construye una instancia stateless.
func NewBlackoutRepository() *BlackoutRepository { return &BlackoutRepository{} }

// Create implementa domain.BlackoutRepository.
func (r *BlackoutRepository) Create(ctx context.Context, in domain.CreateBlackoutInput) (entities.ReservationBlackout, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ReservationBlackout{}, err
	}
	row, err := q.CreateBlackout(ctx, resdb.CreateBlackoutParams{
		CommonAreaID: uuidToPgtype(in.CommonAreaID),
		FromAt:       timeToPgTimestamptz(in.FromAt),
		ToAt:         timeToPgTimestamptz(in.ToAt),
		Reason:       in.Reason,
		CreatedBy:    uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.ReservationBlackout{}, err
	}
	return mapBlackout(row), nil
}

// ListActiveByCommonArea implementa domain.BlackoutRepository.
func (r *BlackoutRepository) ListActiveByCommonArea(ctx context.Context, commonAreaID string) ([]entities.ReservationBlackout, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveBlackoutsByCommonArea(ctx, uuidToPgtype(commonAreaID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.ReservationBlackout, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapBlackout(row))
	}
	return out, nil
}

// ListByCommonAreaAndWindow implementa domain.BlackoutRepository.
func (r *BlackoutRepository) ListByCommonAreaAndWindow(ctx context.Context, commonAreaID string, fromAt, toAt time.Time) ([]entities.ReservationBlackout, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListBlackoutsByCommonAreaAndWindow(ctx,
		uuidToPgtype(commonAreaID),
		timeToPgTimestamptz(fromAt),
		timeToPgTimestamptz(toAt),
	)
	if err != nil {
		return nil, err
	}
	out := make([]entities.ReservationBlackout, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapBlackout(row))
	}
	return out, nil
}

// --- ReservationRepository ---

// ReservationRepository implementa domain.ReservationRepository.
type ReservationRepository struct{}

// NewReservationRepository construye una instancia stateless.
func NewReservationRepository() *ReservationRepository { return &ReservationRepository{} }

// Create implementa domain.ReservationRepository.
func (r *ReservationRepository) Create(ctx context.Context, in domain.CreateReservationInput) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.CreateReservation(ctx, resdb.CreateReservationParams{
		CommonAreaID:      uuidToPgtype(in.CommonAreaID),
		UnitID:            uuidToPgtype(in.UnitID),
		RequestedByUserID: uuidToPgtype(in.RequestedByUserID),
		SlotStartAt:       timeToPgTimestamptz(in.SlotStartAt),
		SlotEndAt:         timeToPgTimestamptz(in.SlotEndAt),
		AttendeesCount:    in.AttendeesCount,
		Cost:              float64ToNumeric(in.Cost),
		SecurityDeposit:   float64ToNumeric(in.SecurityDeposit),
		QrCodeHash:        in.QRCodeHash,
		IdempotencyKey:    in.IdempotencyKey,
		Notes:             in.Notes,
		Status:            string(in.Status),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Reservation{}, domain.ErrReservationSlotConflict
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// GetByID implementa domain.ReservationRepository.
func (r *ReservationRepository) GetByID(ctx context.Context, id string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.GetReservationByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrReservationNotFound
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// GetByIdempotencyKey implementa domain.ReservationRepository.
func (r *ReservationRepository) GetByIdempotencyKey(ctx context.Context, key string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.GetReservationByIdempotencyKey(ctx, &key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrReservationNotFound
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// GetByQRCodeHash implementa domain.ReservationRepository.
func (r *ReservationRepository) GetByQRCodeHash(ctx context.Context, hash string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.GetReservationByQRCodeHash(ctx, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrReservationNotFound
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// List implementa domain.ReservationRepository.
func (r *ReservationRepository) List(ctx context.Context) ([]entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListReservations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Reservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapReservation(row))
	}
	return out, nil
}

// ListByUnit implementa domain.ReservationRepository.
func (r *ReservationRepository) ListByUnit(ctx context.Context, unitID string) ([]entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListReservationsByUnit(ctx, uuidToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Reservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapReservation(row))
	}
	return out, nil
}

// ListByCommonAreaAndDate implementa domain.ReservationRepository.
func (r *ReservationRepository) ListByCommonAreaAndDate(ctx context.Context, commonAreaID string, start, end time.Time) ([]entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListReservationsByCommonAreaAndDate(ctx,
		uuidToPgtype(commonAreaID),
		timeToPgTimestamptz(start),
		timeToPgTimestamptz(end),
	)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Reservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapReservation(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.ReservationRepository.
func (r *ReservationRepository) UpdateStatus(ctx context.Context, id string, expectedVersion int32, newStatus entities.ReservationStatus, actorID string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.UpdateReservationStatus(ctx, resdb.UpdateReservationStatusParams{
		NewStatus:       string(newStatus),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrVersionConflict
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// Approve implementa domain.ReservationRepository.
func (r *ReservationRepository) Approve(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.ApproveReservation(ctx, resdb.ApproveReservationParams{
		ApprovedBy:      uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrVersionConflict
		}
		if isUniqueViolation(err) {
			return entities.Reservation{}, domain.ErrReservationSlotConflict
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// Cancel implementa domain.ReservationRepository.
func (r *ReservationRepository) Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.CancelReservation(ctx, resdb.CancelReservationParams{
		CancelledBy:     uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrVersionConflict
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// Reject implementa domain.ReservationRepository.
func (r *ReservationRepository) Reject(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.RejectReservation(ctx, resdb.RejectReservationParams{
		RejectedBy:      uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrVersionConflict
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// Checkin implementa domain.ReservationRepository.
func (r *ReservationRepository) Checkin(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Reservation{}, err
	}
	row, err := q.CheckinReservation(ctx, resdb.CheckinReservationParams{
		GuardBy:         uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Reservation{}, domain.ErrVersionConflict
		}
		return entities.Reservation{}, err
	}
	return mapReservation(row), nil
}

// --- StatusHistoryRepository ---

// StatusHistoryRepository implementa domain.StatusHistoryRepository.
type StatusHistoryRepository struct{}

// NewStatusHistoryRepository construye una instancia stateless.
func NewStatusHistoryRepository() *StatusHistoryRepository { return &StatusHistoryRepository{} }

// Record implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) Record(ctx context.Context, in domain.RecordStatusHistoryInput) (entities.ReservationStatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ReservationStatusHistory{}, err
	}
	row, err := q.RecordStatusHistory(ctx, resdb.RecordStatusHistoryParams{
		ReservationID: uuidToPgtype(in.ReservationID),
		FromStatus:    in.FromStatus,
		ToStatus:      in.ToStatus,
		ChangedBy:     uuidToPgtypePtr(in.ChangedBy),
		Reason:        in.Reason,
	})
	if err != nil {
		return entities.ReservationStatusHistory{}, err
	}
	return mapStatusHistory(row), nil
}

// ListByReservation implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) ListByReservation(ctx context.Context, reservationID string) ([]entities.ReservationStatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListStatusHistoryByReservation(ctx, uuidToPgtype(reservationID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.ReservationStatusHistory, 0, len(rows))
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
	row, err := q.EnqueueReservationOutboxEvent(ctx, resdb.EnqueueReservationOutboxEventParams{
		AggregateID: uuidToPgtype(in.AggregateID),
		EventType:   string(in.EventType),
		Payload:     in.Payload,
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
	rows, err := q.LockPendingReservationOutboxEvents(ctx, limit)
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
	return q.MarkReservationOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkReservationOutboxEventFailed(ctx, resdb.MarkReservationOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// --- helpers de mapeo ---

func mapCommonArea(r resdb.CommonArea) entities.CommonArea {
	out := entities.CommonArea{
		ID:                  uuidString(r.ID),
		Code:                r.Code,
		Name:                r.Name,
		Kind:                entities.CommonAreaKind(r.Kind),
		MaxCapacity:         r.MaxCapacity,
		OpeningTime:         r.OpeningTime,
		ClosingTime:         r.ClosingTime,
		SlotDurationMinutes: r.SlotDurationMinutes,
		RequiresApproval:    r.RequiresApproval,
		IsActive:            r.IsActive,
		Description:         r.Description,
		Status:              entities.CommonAreaStatus(r.Status),
		CreatedAt:           tsToTime(r.CreatedAt),
		UpdatedAt:           tsToTime(r.UpdatedAt),
		Version:             r.Version,
	}
	if r.CostPerUse.Valid {
		f, err := numericToFloat64(r.CostPerUse)
		if err == nil {
			out.CostPerUse = f
		}
	}
	if r.SecurityDeposit.Valid {
		f, err := numericToFloat64(r.SecurityDeposit)
		if err == nil {
			out.SecurityDeposit = f
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

func mapBlackout(r resdb.ReservationBlackout) entities.ReservationBlackout {
	out := entities.ReservationBlackout{
		ID:           uuidString(r.ID),
		CommonAreaID: uuidString(r.CommonAreaID),
		FromAt:       tsToTime(r.FromAt),
		ToAt:         tsToTime(r.ToAt),
		Reason:       r.Reason,
		Status:       entities.BlackoutStatus(r.Status),
		CreatedAt:    tsToTime(r.CreatedAt),
		UpdatedAt:    tsToTime(r.UpdatedAt),
		Version:      r.Version,
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

func mapReservation(r resdb.Reservation) entities.Reservation {
	out := entities.Reservation{
		ID:                uuidString(r.ID),
		CommonAreaID:      uuidString(r.CommonAreaID),
		UnitID:            uuidString(r.UnitID),
		RequestedByUserID: uuidString(r.RequestedByUserID),
		SlotStartAt:       tsToTime(r.SlotStartAt),
		SlotEndAt:         tsToTime(r.SlotEndAt),
		AttendeesCount:    r.AttendeesCount,
		DepositRefunded:   r.DepositRefunded,
		QRCodeHash:        r.QrCodeHash,
		IdempotencyKey:    r.IdempotencyKey,
		Notes:             r.Notes,
		Status:            entities.ReservationStatus(r.Status),
		CreatedAt:         tsToTime(r.CreatedAt),
		UpdatedAt:         tsToTime(r.UpdatedAt),
		Version:           r.Version,
	}
	if r.Cost.Valid {
		f, err := numericToFloat64(r.Cost)
		if err == nil {
			out.Cost = f
		}
	}
	if r.SecurityDeposit.Valid {
		f, err := numericToFloat64(r.SecurityDeposit)
		if err == nil {
			out.SecurityDeposit = f
		}
	}
	if s := uuidStringPtr(r.ApprovedBy); s != nil {
		out.ApprovedBy = s
	}
	if r.ApprovedAt.Valid {
		t := r.ApprovedAt.Time
		out.ApprovedAt = &t
	}
	if s := uuidStringPtr(r.CancelledBy); s != nil {
		out.CancelledBy = s
	}
	if r.CancelledAt.Valid {
		t := r.CancelledAt.Time
		out.CancelledAt = &t
	}
	if r.ConsumedAt.Valid {
		t := r.ConsumedAt.Time
		out.ConsumedAt = &t
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

func mapStatusHistory(r resdb.ReservationStatusHistory) entities.ReservationStatusHistory {
	out := entities.ReservationStatusHistory{
		ID:            uuidString(r.ID),
		ReservationID: uuidString(r.ReservationID),
		FromStatus:    r.FromStatus,
		ToStatus:      r.ToStatus,
		Reason:        r.Reason,
		ChangedAt:     tsToTime(r.ChangedAt),
	}
	if s := uuidStringPtr(r.ChangedBy); s != nil {
		out.ChangedBy = s
	}
	return out
}

func mapOutbox(r resdb.ReservationsOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:            uuidString(r.ID),
		AggregateID:   uuidString(r.AggregateID),
		EventType:     entities.OutboxEventType(r.EventType),
		Payload:       r.Payload,
		CreatedAt:     tsToTime(r.CreatedAt),
		NextAttemptAt: tsToTime(r.NextAttemptAt),
		Attempts:      r.Attempts,
		LastError:     r.LastError,
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

func numericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, errors.New("numeric is null")
	}
	f64, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f64.Float64, nil
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
	_ domain.CommonAreaRepository    = (*CommonAreaRepository)(nil)
	_ domain.BlackoutRepository      = (*BlackoutRepository)(nil)
	_ domain.ReservationRepository   = (*ReservationRepository)(nil)
	_ domain.StatusHistoryRepository = (*StatusHistoryRepository)(nil)
	_ domain.OutboxRepository        = (*OutboxRepository)(nil)
)
