// Package persistence implementa los puertos del modulo parking usando
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

	"github.com/saas-ph/api/internal/modules/parking/domain"
	"github.com/saas-ph/api/internal/modules/parking/domain/entities"
	parkingdb "github.com/saas-ph/api/internal/modules/parking/infrastructure/persistence/sqlcgen"
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

func querier(ctx context.Context) (*parkingdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return parkingdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("parking: tenant pool is nil")
	}
	return parkingdb.New(t.Pool), nil
}

// --- SpaceRepository ---

// SpaceRepository implementa domain.SpaceRepository.
type SpaceRepository struct{}

// NewSpaceRepository construye una instancia stateless.
func NewSpaceRepository() *SpaceRepository { return &SpaceRepository{} }

// Create implementa domain.SpaceRepository.
func (r *SpaceRepository) Create(ctx context.Context, in domain.CreateSpaceInput) (entities.ParkingSpace, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingSpace{}, err
	}
	row, err := q.CreateParkingSpace(ctx, parkingdb.CreateParkingSpaceParams{
		Code:        in.Code,
		Type:        string(in.Type),
		StructureID: uuidToPgtypePtr(in.StructureID),
		Level:       in.Level,
		Zone:        in.Zone,
		MonthlyFee:  float64ToNumeric(in.MonthlyFee),
		IsVisitor:   in.IsVisitor,
		Notes:       in.Notes,
		CreatedBy:   pgtype.UUID{Valid: false},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.ParkingSpace{}, domain.ErrSpaceCodeDuplicate
		}
		return entities.ParkingSpace{}, err
	}
	return mapSpace(row), nil
}

// GetByID implementa domain.SpaceRepository.
func (r *SpaceRepository) GetByID(ctx context.Context, id string) (entities.ParkingSpace, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingSpace{}, err
	}
	row, err := q.GetParkingSpaceByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ParkingSpace{}, domain.ErrSpaceNotFound
		}
		return entities.ParkingSpace{}, err
	}
	return mapSpace(row), nil
}

// GetByCode implementa domain.SpaceRepository.
func (r *SpaceRepository) GetByCode(ctx context.Context, code string) (entities.ParkingSpace, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingSpace{}, err
	}
	row, err := q.GetParkingSpaceByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ParkingSpace{}, domain.ErrSpaceNotFound
		}
		return entities.ParkingSpace{}, err
	}
	return mapSpace(row), nil
}

// List implementa domain.SpaceRepository.
func (r *SpaceRepository) List(ctx context.Context) ([]entities.ParkingSpace, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListParkingSpaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.ParkingSpace, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapSpace(row))
	}
	return out, nil
}

// Update implementa domain.SpaceRepository.
func (r *SpaceRepository) Update(ctx context.Context, in domain.UpdateSpaceInput) (entities.ParkingSpace, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingSpace{}, err
	}
	row, err := q.UpdateParkingSpace(ctx, parkingdb.UpdateParkingSpaceParams{
		NewCode:         in.Code,
		NewType:         string(in.Type),
		NewStructureID:  uuidToPgtypePtr(in.StructureID),
		NewLevel:        in.Level,
		NewZone:         in.Zone,
		NewMonthlyFee:   float64ToNumeric(in.MonthlyFee),
		NewIsVisitor:    in.IsVisitor,
		NewNotes:        in.Notes,
		NewStatus:       string(in.Status),
		UpdatedBy:       uuidToPgtype(in.ActorID),
		ID:              uuidToPgtype(in.ID),
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ParkingSpace{}, domain.ErrVersionConflict
		}
		if isUniqueViolation(err) {
			return entities.ParkingSpace{}, domain.ErrSpaceCodeDuplicate
		}
		return entities.ParkingSpace{}, err
	}
	return mapSpace(row), nil
}

// SoftDelete implementa domain.SpaceRepository.
func (r *SpaceRepository) SoftDelete(ctx context.Context, id string, expectedVersion int32, actorID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.SoftDeleteParkingSpace(ctx, parkingdb.SoftDeleteParkingSpaceParams{
		DeletedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// --- AssignmentRepository ---

// AssignmentRepository implementa domain.AssignmentRepository.
type AssignmentRepository struct{}

// NewAssignmentRepository construye una instancia stateless.
func NewAssignmentRepository() *AssignmentRepository { return &AssignmentRepository{} }

// Create implementa domain.AssignmentRepository.
func (r *AssignmentRepository) Create(ctx context.Context, in domain.CreateAssignmentInput) (entities.ParkingAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingAssignment{}, err
	}
	row, err := q.CreateParkingAssignment(ctx, parkingdb.CreateParkingAssignmentParams{
		ParkingSpaceID:   uuidToPgtype(in.ParkingSpaceID),
		UnitID:           uuidToPgtype(in.UnitID),
		VehicleID:        uuidToPgtypePtr(in.VehicleID),
		AssignedByUserID: uuidToPgtypePtr(in.AssignedByUserID),
		SinceDate:        timeToPgDate(in.SinceDate),
		Notes:            in.Notes,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.ParkingAssignment{}, domain.ErrAssignmentAlreadyActive
		}
		return entities.ParkingAssignment{}, err
	}
	return mapAssignment(row), nil
}

// GetByID implementa domain.AssignmentRepository.
func (r *AssignmentRepository) GetByID(ctx context.Context, id string) (entities.ParkingAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingAssignment{}, err
	}
	row, err := q.GetParkingAssignmentByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
		}
		return entities.ParkingAssignment{}, err
	}
	return mapAssignment(row), nil
}

// GetActiveBySpaceID implementa domain.AssignmentRepository.
func (r *AssignmentRepository) GetActiveBySpaceID(ctx context.Context, spaceID string) (entities.ParkingAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingAssignment{}, err
	}
	row, err := q.GetActiveAssignmentBySpaceID(ctx, uuidToPgtype(spaceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ParkingAssignment{}, domain.ErrAssignmentNotFound
		}
		return entities.ParkingAssignment{}, err
	}
	return mapAssignment(row), nil
}

// ListActiveByUnitID implementa domain.AssignmentRepository.
func (r *AssignmentRepository) ListActiveByUnitID(ctx context.Context, unitID string) ([]entities.ParkingAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveAssignmentsByUnitID(ctx, uuidToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.ParkingAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAssignment(row))
	}
	return out, nil
}

// ListBySpaceID implementa domain.AssignmentRepository.
func (r *AssignmentRepository) ListBySpaceID(ctx context.Context, spaceID string) ([]entities.ParkingAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssignmentsBySpaceID(ctx, uuidToPgtype(spaceID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.ParkingAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAssignment(row))
	}
	return out, nil
}

// CloseAssignment implementa domain.AssignmentRepository.
func (r *AssignmentRepository) CloseAssignment(ctx context.Context, id string, untilDate time.Time, expectedVersion int32, actorID string) (entities.ParkingAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ParkingAssignment{}, err
	}
	row, err := q.CloseAssignment(ctx, parkingdb.CloseAssignmentParams{
		UntilDate:       timeToPgDate(untilDate),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ParkingAssignment{}, domain.ErrVersionConflict
		}
		return entities.ParkingAssignment{}, err
	}
	return mapAssignment(row), nil
}

// SoftDelete implementa domain.AssignmentRepository.
func (r *AssignmentRepository) SoftDelete(ctx context.Context, id string, expectedVersion int32, actorID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.SoftDeleteAssignment(ctx, parkingdb.SoftDeleteAssignmentParams{
		DeletedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// --- AssignmentHistoryRepository ---

// AssignmentHistoryRepository implementa domain.AssignmentHistoryRepository.
type AssignmentHistoryRepository struct{}

// NewAssignmentHistoryRepository construye una instancia stateless.
func NewAssignmentHistoryRepository() *AssignmentHistoryRepository {
	return &AssignmentHistoryRepository{}
}

// Record implementa domain.AssignmentHistoryRepository.
func (r *AssignmentHistoryRepository) Record(ctx context.Context, in domain.RecordHistoryInput) (entities.AssignmentHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssignmentHistory{}, err
	}
	row, err := q.RecordAssignmentHistory(ctx, parkingdb.RecordAssignmentHistoryParams{
		ParkingSpaceID:  uuidToPgtype(in.ParkingSpaceID),
		UnitID:          uuidToPgtype(in.UnitID),
		AssignmentID:    uuidToPgtypePtr(in.AssignmentID),
		SinceDate:       timeToPgDate(in.SinceDate),
		UntilDate:       timePtrToPgDate(in.UntilDate),
		ClosedReason:    in.ClosedReason,
		SnapshotPayload: in.SnapshotPayload,
		RecordedBy:      uuidToPgtypePtr(in.RecordedBy),
	})
	if err != nil {
		return entities.AssignmentHistory{}, err
	}
	return mapHistory(row), nil
}

// ListBySpaceID implementa domain.AssignmentHistoryRepository.
func (r *AssignmentHistoryRepository) ListBySpaceID(ctx context.Context, spaceID string) ([]entities.AssignmentHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssignmentHistoryBySpaceID(ctx, uuidToPgtype(spaceID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.AssignmentHistory, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapHistory(row))
	}
	return out, nil
}

// ListByUnitID implementa domain.AssignmentHistoryRepository.
func (r *AssignmentHistoryRepository) ListByUnitID(ctx context.Context, unitID string) ([]entities.AssignmentHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssignmentHistoryByUnitID(ctx, uuidToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.AssignmentHistory, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapHistory(row))
	}
	return out, nil
}

// --- VisitorReservationRepository ---

// VisitorReservationRepository implementa domain.VisitorReservationRepository.
type VisitorReservationRepository struct{}

// NewVisitorReservationRepository construye una instancia stateless.
func NewVisitorReservationRepository() *VisitorReservationRepository {
	return &VisitorReservationRepository{}
}

// Create implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) Create(ctx context.Context, in domain.CreateReservationInput) (entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorReservation{}, err
	}
	row, err := q.CreateVisitorReservation(ctx, parkingdb.CreateVisitorReservationParams{
		ParkingSpaceID:  uuidToPgtype(in.ParkingSpaceID),
		UnitID:          uuidToPgtype(in.UnitID),
		RequestedBy:     uuidToPgtype(in.RequestedBy),
		VisitorName:     in.VisitorName,
		VisitorDocument: in.VisitorDocument,
		VehiclePlate:    in.VehiclePlate,
		SlotStartAt:     timeToPgTimestamptz(in.SlotStartAt),
		SlotEndAt:       timeToPgTimestamptz(in.SlotEndAt),
		IdempotencyKey:  in.IdempotencyKey,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.VisitorReservation{}, domain.ErrReservationSlotConflict
		}
		return entities.VisitorReservation{}, err
	}
	return mapReservation(row), nil
}

// GetByID implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) GetByID(ctx context.Context, id string) (entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorReservation{}, err
	}
	row, err := q.GetVisitorReservationByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.VisitorReservation{}, domain.ErrReservationNotFound
		}
		return entities.VisitorReservation{}, err
	}
	return mapReservation(row), nil
}

// ListByDate implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) ListByDate(ctx context.Context, start, end time.Time) ([]entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListVisitorReservationsByDate(ctx, timeToPgTimestamptz(start), timeToPgTimestamptz(end))
	if err != nil {
		return nil, err
	}
	out := make([]entities.VisitorReservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapReservation(row))
	}
	return out, nil
}

// ListByUnit implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) ListByUnit(ctx context.Context, unitID string) ([]entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListVisitorReservationsByUnit(ctx, uuidToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.VisitorReservation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapReservation(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) UpdateStatus(ctx context.Context, id string, expectedVersion int32, newStatus entities.ReservationStatus, actorID string) (entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorReservation{}, err
	}
	row, err := q.UpdateVisitorReservationStatus(ctx, parkingdb.UpdateVisitorReservationStatusParams{
		NewStatus:       string(newStatus),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.VisitorReservation{}, domain.ErrVersionConflict
		}
		return entities.VisitorReservation{}, err
	}
	return mapReservation(row), nil
}

// Cancel implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorReservation{}, err
	}
	row, err := q.CancelVisitorReservation(ctx, parkingdb.CancelVisitorReservationParams{
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.VisitorReservation{}, domain.ErrVersionConflict
		}
		return entities.VisitorReservation{}, err
	}
	return mapReservation(row), nil
}

// GetByIdempotencyKey implementa domain.VisitorReservationRepository.
func (r *VisitorReservationRepository) GetByIdempotencyKey(ctx context.Context, key string) (entities.VisitorReservation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorReservation{}, err
	}
	row, err := q.GetVisitorReservationByIdempotencyKey(ctx, &key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.VisitorReservation{}, domain.ErrReservationNotFound
		}
		return entities.VisitorReservation{}, err
	}
	return mapReservation(row), nil
}

// --- LotteryRunRepository ---

// LotteryRunRepository implementa domain.LotteryRunRepository.
type LotteryRunRepository struct{}

// NewLotteryRunRepository construye una instancia stateless.
func NewLotteryRunRepository() *LotteryRunRepository { return &LotteryRunRepository{} }

// Create implementa domain.LotteryRunRepository.
func (r *LotteryRunRepository) Create(ctx context.Context, in domain.CreateLotteryRunInput) (entities.LotteryRun, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.LotteryRun{}, err
	}
	row, err := q.CreateLotteryRun(ctx, parkingdb.CreateLotteryRunParams{
		Name:       in.Name,
		SeedHash:   in.SeedHash,
		Criteria:   in.Criteria,
		ExecutedBy: uuidToPgtype(in.ExecutedBy),
	})
	if err != nil {
		return entities.LotteryRun{}, err
	}
	return mapLotteryRun(row), nil
}

// GetByID implementa domain.LotteryRunRepository.
func (r *LotteryRunRepository) GetByID(ctx context.Context, id string) (entities.LotteryRun, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.LotteryRun{}, err
	}
	row, err := q.GetLotteryRunByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.LotteryRun{}, domain.ErrLotteryNotFound
		}
		return entities.LotteryRun{}, err
	}
	return mapLotteryRun(row), nil
}

// List implementa domain.LotteryRunRepository.
func (r *LotteryRunRepository) List(ctx context.Context) ([]entities.LotteryRun, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListLotteryRuns(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.LotteryRun, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapLotteryRun(row))
	}
	return out, nil
}

// --- LotteryResultRepository ---

// LotteryResultRepository implementa domain.LotteryResultRepository.
type LotteryResultRepository struct{}

// NewLotteryResultRepository construye una instancia stateless.
func NewLotteryResultRepository() *LotteryResultRepository { return &LotteryResultRepository{} }

// CreateBatch implementa domain.LotteryResultRepository.
func (r *LotteryResultRepository) CreateBatch(ctx context.Context, results []domain.CreateLotteryResultInput) ([]entities.LotteryResult, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.LotteryResult, 0, len(results))
	for _, in := range results {
		row, rErr := q.CreateLotteryResult(ctx, parkingdb.CreateLotteryResultParams{
			LotteryRunID:   uuidToPgtype(in.LotteryRunID),
			UnitID:         uuidToPgtype(in.UnitID),
			ParkingSpaceID: uuidToPgtypePtr(in.ParkingSpaceID),
			Position:       in.Position,
			Status:         string(in.Status),
			CreatedBy:      pgtype.UUID{Valid: false},
		})
		if rErr != nil {
			return nil, rErr
		}
		out = append(out, mapLotteryResult(row))
	}
	return out, nil
}

// ListByRunID implementa domain.LotteryResultRepository.
func (r *LotteryResultRepository) ListByRunID(ctx context.Context, runID string) ([]entities.LotteryResult, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListLotteryResultsByRunID(ctx, uuidToPgtype(runID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.LotteryResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapLotteryResult(row))
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
	row, err := q.EnqueueParkingOutboxEvent(ctx, parkingdb.EnqueueParkingOutboxEventParams{
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
	rows, err := q.LockPendingParkingOutboxEvents(ctx, limit)
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
	return q.MarkParkingOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkParkingOutboxEventFailed(ctx, parkingdb.MarkParkingOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// --- helpers de mapeo ---

func mapSpace(r parkingdb.ParkingSpace) entities.ParkingSpace {
	out := entities.ParkingSpace{
		ID:        uuidString(r.ID),
		Code:      r.Code,
		Type:      entities.SpaceType(r.Type),
		Level:     r.Level,
		Zone:      r.Zone,
		IsVisitor: r.IsVisitor,
		Notes:     r.Notes,
		Status:    entities.SpaceStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
	}
	if s := uuidStringPtr(r.StructureID); s != nil {
		out.StructureID = s
	}
	if r.MonthlyFee.Valid {
		f, err := numericToFloat64(r.MonthlyFee)
		if err == nil {
			out.MonthlyFee = &f
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

func mapAssignment(r parkingdb.ParkingAssignment) entities.ParkingAssignment {
	out := entities.ParkingAssignment{
		ID:             uuidString(r.ID),
		ParkingSpaceID: uuidString(r.ParkingSpaceID),
		UnitID:         uuidString(r.UnitID),
		Notes:          r.Notes,
		Status:         entities.AssignmentStatus(r.Status),
		SinceDate:      dateToTime(r.SinceDate),
		CreatedAt:      tsToTime(r.CreatedAt),
		UpdatedAt:      tsToTime(r.UpdatedAt),
		Version:        r.Version,
	}
	if s := uuidStringPtr(r.VehicleID); s != nil {
		out.VehicleID = s
	}
	if s := uuidStringPtr(r.AssignedByUserID); s != nil {
		out.AssignedByUserID = s
	}
	if r.UntilDate.Valid {
		t := time.Date(r.UntilDate.Time.Year(), r.UntilDate.Time.Month(), r.UntilDate.Time.Day(), 0, 0, 0, 0, time.UTC)
		out.UntilDate = &t
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

func mapHistory(r parkingdb.ParkingAssignmentHistory) entities.AssignmentHistory {
	out := entities.AssignmentHistory{
		ID:              uuidString(r.ID),
		ParkingSpaceID:  uuidString(r.ParkingSpaceID),
		UnitID:          uuidString(r.UnitID),
		SinceDate:       dateToTime(r.SinceDate),
		ClosedReason:    r.ClosedReason,
		SnapshotPayload: r.SnapshotPayload,
		RecordedAt:      tsToTime(r.RecordedAt),
	}
	if s := uuidStringPtr(r.AssignmentID); s != nil {
		out.AssignmentID = s
	}
	if r.UntilDate.Valid {
		t := time.Date(r.UntilDate.Time.Year(), r.UntilDate.Time.Month(), r.UntilDate.Time.Day(), 0, 0, 0, 0, time.UTC)
		out.UntilDate = &t
	}
	if s := uuidStringPtr(r.RecordedBy); s != nil {
		out.RecordedBy = s
	}
	return out
}

func mapReservation(r parkingdb.ParkingVisitorReservation) entities.VisitorReservation {
	out := entities.VisitorReservation{
		ID:              uuidString(r.ID),
		ParkingSpaceID:  uuidString(r.ParkingSpaceID),
		UnitID:          uuidString(r.UnitID),
		RequestedBy:     uuidString(r.RequestedBy),
		VisitorName:     r.VisitorName,
		VisitorDocument: r.VisitorDocument,
		VehiclePlate:    r.VehiclePlate,
		SlotStartAt:     tsToTime(r.SlotStartAt),
		SlotEndAt:       tsToTime(r.SlotEndAt),
		IdempotencyKey:  r.IdempotencyKey,
		Status:          entities.ReservationStatus(r.Status),
		CreatedAt:       tsToTime(r.CreatedAt),
		UpdatedAt:       tsToTime(r.UpdatedAt),
		Version:         r.Version,
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

func mapLotteryRun(r parkingdb.ParkingLotteryRun) entities.LotteryRun {
	out := entities.LotteryRun{
		ID:         uuidString(r.ID),
		Name:       r.Name,
		SeedHash:   r.SeedHash,
		Criteria:   r.Criteria,
		ExecutedAt: tsToTime(r.ExecutedAt),
		ExecutedBy: uuidString(r.ExecutedBy),
		Status:     entities.LotteryStatus(r.Status),
		CreatedAt:  tsToTime(r.CreatedAt),
		UpdatedAt:  tsToTime(r.UpdatedAt),
		Version:    r.Version,
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

func mapLotteryResult(r parkingdb.ParkingLotteryResult) entities.LotteryResult {
	out := entities.LotteryResult{
		ID:           uuidString(r.ID),
		LotteryRunID: uuidString(r.LotteryRunID),
		UnitID:       uuidString(r.UnitID),
		Position:     r.Position,
		Status:       entities.LotteryResultStatus(r.Status),
		CreatedAt:    tsToTime(r.CreatedAt),
		UpdatedAt:    tsToTime(r.UpdatedAt),
		Version:      r.Version,
	}
	if s := uuidStringPtr(r.ParkingSpaceID); s != nil {
		out.ParkingSpaceID = s
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

func mapOutbox(r parkingdb.ParkingOutboxEvent) entities.OutboxEvent {
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

func dateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

func timeToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func timePtrToPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
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

func float64ToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	var n pgtype.Numeric
	if err := n.Scan(*f); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
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
		var f float64
		_, scanErr := time.ParseDuration("0") // dummy
		_ = scanErr
		// Use pgtype scan back to float
		var num pgtype.Numeric
		if sErr := num.Scan(val); sErr != nil {
			return 0, sErr
		}
		f64, f64Err := num.Float64Value()
		if f64Err != nil {
			return 0, f64Err
		}
		f = f64.Float64
		return f, nil
	default:
		return 0, errors.New("unexpected numeric type")
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps postgres errors; check for SQLSTATE 23505.
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

// Compile-time checks: each repo implements the domain port.
var (
	_ domain.SpaceRepository              = (*SpaceRepository)(nil)
	_ domain.AssignmentRepository         = (*AssignmentRepository)(nil)
	_ domain.AssignmentHistoryRepository  = (*AssignmentHistoryRepository)(nil)
	_ domain.VisitorReservationRepository = (*VisitorReservationRepository)(nil)
	_ domain.LotteryRunRepository         = (*LotteryRunRepository)(nil)
	_ domain.LotteryResultRepository      = (*LotteryResultRepository)(nil)
	_ domain.OutboxRepository             = (*OutboxRepository)(nil)
)
