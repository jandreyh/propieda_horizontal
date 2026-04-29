// Package persistence implementa los puertos del modulo pqrs usando
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

	"github.com/saas-ph/api/internal/modules/pqrs/domain"
	"github.com/saas-ph/api/internal/modules/pqrs/domain/entities"
	pqrsdb "github.com/saas-ph/api/internal/modules/pqrs/infrastructure/persistence/sqlcgen"
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

func querier(ctx context.Context) (*pqrsdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return pqrsdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("pqrs: tenant pool is nil")
	}
	return pqrsdb.New(t.Pool), nil
}

// --- CategoryRepository ---

// CategoryRepository implementa domain.CategoryRepository.
type CategoryRepository struct{}

// NewCategoryRepository construye una instancia stateless.
func NewCategoryRepository() *CategoryRepository { return &CategoryRepository{} }

// Create implementa domain.CategoryRepository.
func (r *CategoryRepository) Create(ctx context.Context, in domain.CreateCategoryInput) (entities.Category, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Category{}, err
	}
	row, err := q.CreatePQRSCategory(ctx, pqrsdb.CreatePQRSCategoryParams{
		Code:                  in.Code,
		Name:                  in.Name,
		DefaultAssigneeRoleID: uuidToPgtypePtr(in.DefaultAssigneeRoleID),
		CreatedBy:             uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Category{}, domain.ErrCategoryCodeDuplicate
		}
		return entities.Category{}, err
	}
	return mapCategory(row), nil
}

// GetByID implementa domain.CategoryRepository.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (entities.Category, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Category{}, err
	}
	row, err := q.GetPQRSCategoryByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Category{}, domain.ErrCategoryNotFound
		}
		return entities.Category{}, err
	}
	return mapCategory(row), nil
}

// List implementa domain.CategoryRepository.
func (r *CategoryRepository) List(ctx context.Context) ([]entities.Category, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPQRSCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCategory(row))
	}
	return out, nil
}

// Update implementa domain.CategoryRepository.
func (r *CategoryRepository) Update(ctx context.Context, in domain.UpdateCategoryInput) (entities.Category, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Category{}, err
	}
	row, err := q.UpdatePQRSCategory(ctx, pqrsdb.UpdatePQRSCategoryParams{
		NewCode:                  in.Code,
		NewName:                  in.Name,
		NewDefaultAssigneeRoleID: uuidToPgtypePtr(in.DefaultAssigneeRoleID),
		NewStatus:                string(in.Status),
		UpdatedBy:                uuidToPgtype(in.ActorID),
		ID:                       uuidToPgtype(in.ID),
		ExpectedVersion:          in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Category{}, domain.ErrVersionConflict
		}
		if isUniqueViolation(err) {
			return entities.Category{}, domain.ErrCategoryCodeDuplicate
		}
		return entities.Category{}, err
	}
	return mapCategory(row), nil
}

// --- TicketRepository ---

// TicketRepository implementa domain.TicketRepository.
type TicketRepository struct{}

// NewTicketRepository construye una instancia stateless.
func NewTicketRepository() *TicketRepository { return &TicketRepository{} }

// NextSerialNumber implementa domain.TicketRepository.
func (r *TicketRepository) NextSerialNumber(ctx context.Context, year int32) (int32, error) {
	q, err := querier(ctx)
	if err != nil {
		return 0, err
	}
	serial, err := q.NextPQRSSerialNumber(ctx, year)
	if err != nil {
		return 0, err
	}
	return serial, nil
}

// Create implementa domain.TicketRepository.
func (r *TicketRepository) Create(ctx context.Context, in domain.CreateTicketInput) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.CreatePQRSTicket(ctx, pqrsdb.CreatePQRSTicketParams{
		TicketYear:      in.TicketYear,
		SerialNumber:    in.SerialNumber,
		PqrType:         string(in.PQRType),
		CategoryID:      uuidToPgtypePtr(in.CategoryID),
		Subject:         in.Subject,
		Body:            in.Body,
		RequesterUserID: uuidToPgtype(in.RequesterUserID),
		IsAnonymous:     in.IsAnonymous,
		SlaDueAt:        timePtrToPgTimestamptz(in.SLADueAt),
		CreatedBy:       uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// GetByID implementa domain.TicketRepository.
func (r *TicketRepository) GetByID(ctx context.Context, id string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.GetPQRSTicketByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrTicketNotFound
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// List implementa domain.TicketRepository.
func (r *TicketRepository) List(ctx context.Context, filter domain.TicketListFilter) ([]entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}

	params := pqrsdb.ListPQRSTicketsParams{}
	if filter.Status != nil {
		s := string(*filter.Status)
		params.FilterStatus = &s
	}
	if filter.PQRType != nil {
		s := string(*filter.PQRType)
		params.FilterPqrType = &s
	}
	if filter.RequesterUserID != nil {
		params.FilterRequester = uuidToPgtype(*filter.RequesterUserID)
	}
	if filter.AssignedToUserID != nil {
		params.FilterAssigned = uuidToPgtype(*filter.AssignedToUserID)
	}

	rows, err := q.ListPQRSTickets(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Ticket, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTicket(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.TicketRepository.
func (r *TicketRepository) UpdateStatus(ctx context.Context, id string, newStatus entities.TicketStatus, expectedVersion int32, actorID string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.UpdatePQRSTicketStatus(ctx, pqrsdb.UpdatePQRSTicketStatusParams{
		NewStatus:       string(newStatus),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrVersionConflict
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// Assign implementa domain.TicketRepository.
func (r *TicketRepository) Assign(ctx context.Context, id string, assigneeUserID string, expectedVersion int32, actorID string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.AssignPQRSTicket(ctx, pqrsdb.AssignPQRSTicketParams{
		AssigneeUserID:  uuidToPgtype(assigneeUserID),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrVersionConflict
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// SetResponded implementa domain.TicketRepository.
func (r *TicketRepository) SetResponded(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.SetPQRSTicketResponded(ctx, pqrsdb.SetPQRSTicketRespondedParams{
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrVersionConflict
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// Close implementa domain.TicketRepository.
func (r *TicketRepository) Close(ctx context.Context, id string, rating *int32, feedback *string, expectedVersion int32, actorID string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	ratingPg := pgtype.Int4{Valid: false}
	if rating != nil {
		ratingPg = pgtype.Int4{Int32: *rating, Valid: true}
	}
	row, err := q.ClosePQRSTicket(ctx, pqrsdb.ClosePQRSTicketParams{
		Rating:          ratingPg,
		Feedback:        feedback,
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrVersionConflict
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// Escalate implementa domain.TicketRepository.
func (r *TicketRepository) Escalate(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.EscalatePQRSTicket(ctx, pqrsdb.EscalatePQRSTicketParams{
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrVersionConflict
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// Cancel implementa domain.TicketRepository.
func (r *TicketRepository) Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Ticket, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ticket{}, err
	}
	row, err := q.CancelPQRSTicket(ctx, pqrsdb.CancelPQRSTicketParams{
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Ticket{}, domain.ErrVersionConflict
		}
		return entities.Ticket{}, err
	}
	return mapTicket(row), nil
}

// --- ResponseRepository ---

// ResponseRepository implementa domain.ResponseRepository.
type ResponseRepository struct{}

// NewResponseRepository construye una instancia stateless.
func NewResponseRepository() *ResponseRepository { return &ResponseRepository{} }

// Create implementa domain.ResponseRepository.
func (r *ResponseRepository) Create(ctx context.Context, in domain.CreateResponseInput) (entities.Response, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Response{}, err
	}
	row, err := q.CreatePQRSResponse(ctx, pqrsdb.CreatePQRSResponseParams{
		TicketID:          uuidToPgtype(in.TicketID),
		ResponseType:      string(in.ResponseType),
		Body:              in.Body,
		RespondedByUserID: uuidToPgtype(in.RespondedByUserID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Response{}, domain.ErrOfficialResponseExists
		}
		return entities.Response{}, err
	}
	return mapResponse(row), nil
}

// ListByTicketID implementa domain.ResponseRepository.
func (r *ResponseRepository) ListByTicketID(ctx context.Context, ticketID string) ([]entities.Response, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPQRSResponsesByTicketID(ctx, uuidToPgtype(ticketID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Response, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapResponse(row))
	}
	return out, nil
}

// HasOfficialResponse implementa domain.ResponseRepository.
func (r *ResponseRepository) HasOfficialResponse(ctx context.Context, ticketID string) (bool, error) {
	q, err := querier(ctx)
	if err != nil {
		return false, err
	}
	return q.HasPQRSOfficialResponse(ctx, uuidToPgtype(ticketID))
}

// --- StatusHistoryRepository ---

// StatusHistoryRepository implementa domain.StatusHistoryRepository.
type StatusHistoryRepository struct{}

// NewStatusHistoryRepository construye una instancia stateless.
func NewStatusHistoryRepository() *StatusHistoryRepository {
	return &StatusHistoryRepository{}
}

// Record implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) Record(ctx context.Context, in domain.RecordHistoryInput) (entities.StatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.StatusHistory{}, err
	}
	row, err := q.RecordPQRSStatusHistory(ctx, pqrsdb.RecordPQRSStatusHistoryParams{
		TicketID:             uuidToPgtype(in.TicketID),
		FromStatus:           in.FromStatus,
		ToStatus:             in.ToStatus,
		TransitionedByUserID: uuidToPgtype(in.TransitionedByUserID),
		Notes:                in.Notes,
	})
	if err != nil {
		return entities.StatusHistory{}, err
	}
	return mapStatusHistory(row), nil
}

// ListByTicketID implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) ListByTicketID(ctx context.Context, ticketID string) ([]entities.StatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPQRSStatusHistoryByTicketID(ctx, uuidToPgtype(ticketID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.StatusHistory, 0, len(rows))
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
	row, err := q.EnqueuePQRSOutboxEvent(ctx, pqrsdb.EnqueuePQRSOutboxEventParams{
		TicketID:       uuidToPgtype(in.TicketID),
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
	rows, err := q.LockPendingPQRSOutboxEvents(ctx, limit)
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
	return q.MarkPQRSOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkPQRSOutboxEventFailed(ctx, pqrsdb.MarkPQRSOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// --- helpers de mapeo ---

func mapCategory(r pqrsdb.PqrsCategory) entities.Category {
	out := entities.Category{
		ID:        uuidString(r.ID),
		Code:      r.Code,
		Name:      r.Name,
		Status:    entities.CategoryStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
	}
	if s := uuidStringPtr(r.DefaultAssigneeRoleID); s != nil {
		out.DefaultAssigneeRoleID = s
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

func mapTicket(r pqrsdb.PqrsTicket) entities.Ticket {
	out := entities.Ticket{
		ID:                uuidString(r.ID),
		TicketYear:        r.TicketYear,
		SerialNumber:      r.SerialNumber,
		PQRType:           entities.PQRType(r.PqrType),
		Subject:           r.Subject,
		Body:              r.Body,
		RequesterUserID:   uuidString(r.RequesterUserID),
		IsAnonymous:       r.IsAnonymous,
		RequesterFeedback: r.RequesterFeedback,
		Status:            entities.TicketStatus(r.Status),
		CreatedAt:         tsToTime(r.CreatedAt),
		UpdatedAt:         tsToTime(r.UpdatedAt),
		Version:           r.Version,
	}
	if s := uuidStringPtr(r.CategoryID); s != nil {
		out.CategoryID = s
	}
	if s := uuidStringPtr(r.AssignedToUserID); s != nil {
		out.AssignedToUserID = s
	}
	if r.AssignedAt.Valid {
		t := r.AssignedAt.Time
		out.AssignedAt = &t
	}
	if r.RespondedAt.Valid {
		t := r.RespondedAt.Time
		out.RespondedAt = &t
	}
	if r.ClosedAt.Valid {
		t := r.ClosedAt.Time
		out.ClosedAt = &t
	}
	if r.EscalatedAt.Valid {
		t := r.EscalatedAt.Time
		out.EscalatedAt = &t
	}
	if r.CancelledAt.Valid {
		t := r.CancelledAt.Time
		out.CancelledAt = &t
	}
	if r.SlaDueAt.Valid {
		t := r.SlaDueAt.Time
		out.SLADueAt = &t
	}
	if r.RequesterRating.Valid {
		v := r.RequesterRating.Int32
		out.RequesterRating = &v
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

func mapResponse(r pqrsdb.PqrsResponse) entities.Response {
	out := entities.Response{
		ID:                uuidString(r.ID),
		TicketID:          uuidString(r.TicketID),
		ResponseType:      entities.ResponseType(r.ResponseType),
		Body:              r.Body,
		RespondedByUserID: uuidString(r.RespondedByUserID),
		RespondedAt:       tsToTime(r.RespondedAt),
		Status:            r.Status,
		CreatedAt:         tsToTime(r.CreatedAt),
		UpdatedAt:         tsToTime(r.UpdatedAt),
		Version:           r.Version,
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

func mapStatusHistory(r pqrsdb.PqrsStatusHistory) entities.StatusHistory {
	out := entities.StatusHistory{
		ID:                   uuidString(r.ID),
		TicketID:             uuidString(r.TicketID),
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

func mapOutbox(r pqrsdb.PqrsOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:             uuidString(r.ID),
		TicketID:       uuidString(r.TicketID),
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

func timePtrToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
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
	_ domain.CategoryRepository      = (*CategoryRepository)(nil)
	_ domain.TicketRepository        = (*TicketRepository)(nil)
	_ domain.ResponseRepository      = (*ResponseRepository)(nil)
	_ domain.StatusHistoryRepository = (*StatusHistoryRepository)(nil)
	_ domain.OutboxRepository        = (*OutboxRepository)(nil)
)
