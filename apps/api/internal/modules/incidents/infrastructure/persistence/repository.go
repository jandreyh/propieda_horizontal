// Package persistence implementa los puertos del modulo incidents usando
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

	"github.com/saas-ph/api/internal/modules/incidents/domain"
	"github.com/saas-ph/api/internal/modules/incidents/domain/entities"
	incidentsdb "github.com/saas-ph/api/internal/modules/incidents/infrastructure/persistence/sqlcgen"
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

func querier(ctx context.Context) (*incidentsdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return incidentsdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("incidents: tenant pool is nil")
	}
	return incidentsdb.New(t.Pool), nil
}

// --- IncidentRepository ---

// IncidentRepository implementa domain.IncidentRepository.
type IncidentRepository struct{}

// NewIncidentRepository construye una instancia stateless.
func NewIncidentRepository() *IncidentRepository { return &IncidentRepository{} }

// Create implementa domain.IncidentRepository.
func (r *IncidentRepository) Create(ctx context.Context, in domain.CreateIncidentInput) (entities.Incident, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Incident{}, err
	}
	row, err := q.CreateIncident(ctx, incidentsdb.CreateIncidentParams{
		IncidentType:     string(in.IncidentType),
		Severity:         string(in.Severity),
		Title:            in.Title,
		Description:      in.Description,
		ReportedByUserID: uuidToPgtype(in.ReportedByUserID),
		ReportedAt:       timeToPgTimestamptz(in.ReportedAt),
		StructureID:      uuidToPgtypePtr(in.StructureID),
		LocationDetail:   in.LocationDetail,
		SlaAssignDueAt:   timePtrToPgTimestamptz(in.SLAAssignDueAt),
		SlaResolveDueAt:  timePtrToPgTimestamptz(in.SLAResolveDueAt),
	})
	if err != nil {
		return entities.Incident{}, err
	}
	return mapIncident(row), nil
}

// GetByID implementa domain.IncidentRepository.
func (r *IncidentRepository) GetByID(ctx context.Context, id string) (entities.Incident, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Incident{}, err
	}
	row, err := q.GetIncidentByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Incident{}, domain.ErrIncidentNotFound
		}
		return entities.Incident{}, err
	}
	return mapIncident(row), nil
}

// List implementa domain.IncidentRepository.
func (r *IncidentRepository) List(ctx context.Context, filter domain.IncidentListFilter) ([]entities.Incident, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}

	var rows []incidentsdb.Incident

	switch {
	case filter.Status != nil:
		rows, err = q.ListIncidentsByStatus(ctx, string(*filter.Status))
	case filter.Severity != nil:
		rows, err = q.ListIncidentsBySeverity(ctx, string(*filter.Severity))
	case filter.ReportedByUserID != nil:
		rows, err = q.ListIncidentsByReporter(ctx, uuidToPgtype(*filter.ReportedByUserID))
	default:
		rows, err = q.ListIncidents(ctx)
	}
	if err != nil {
		return nil, err
	}

	out := make([]entities.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapIncident(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.IncidentRepository.
func (r *IncidentRepository) UpdateStatus(ctx context.Context, in domain.UpdateIncidentStatusInput) (entities.Incident, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Incident{}, err
	}
	row, err := q.UpdateIncidentStatus(ctx, incidentsdb.UpdateIncidentStatusParams{
		NewStatus:           string(in.NewStatus),
		NewAssignedToUserID: uuidToPgtypePtr(in.AssignedToUserID),
		NewAssignedAt:       timePtrToPgTimestamptz(in.AssignedAt),
		NewStartedAt:        timePtrToPgTimestamptz(in.StartedAt),
		NewResolvedAt:       timePtrToPgTimestamptz(in.ResolvedAt),
		NewClosedAt:         timePtrToPgTimestamptz(in.ClosedAt),
		NewCancelledAt:      timePtrToPgTimestamptz(in.CancelledAt),
		NewResolutionNotes:  in.ResolutionNotes,
		NewEscalated:        in.Escalated,
		UpdatedBy:           uuidToPgtype(in.ActorID),
		ID:                  uuidToPgtype(in.ID),
		ExpectedVersion:     in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Incident{}, domain.ErrVersionConflict
		}
		return entities.Incident{}, err
	}
	return mapIncident(row), nil
}

// --- AttachmentRepository ---

// AttachmentRepository implementa domain.AttachmentRepository.
type AttachmentRepository struct{}

// NewAttachmentRepository construye una instancia stateless.
func NewAttachmentRepository() *AttachmentRepository { return &AttachmentRepository{} }

// Create implementa domain.AttachmentRepository.
func (r *AttachmentRepository) Create(ctx context.Context, in domain.CreateAttachmentInput) (entities.Attachment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Attachment{}, err
	}
	row, err := q.CreateIncidentAttachment(ctx, incidentsdb.CreateIncidentAttachmentParams{
		IncidentID: uuidToPgtype(in.IncidentID),
		Url:        in.URL,
		MimeType:   in.MimeType,
		SizeBytes:  in.SizeBytes,
		UploadedBy: uuidToPgtype(in.UploadedBy),
	})
	if err != nil {
		return entities.Attachment{}, err
	}
	return mapAttachment(row), nil
}

// ListByIncidentID implementa domain.AttachmentRepository.
func (r *AttachmentRepository) ListByIncidentID(ctx context.Context, incidentID string) ([]entities.Attachment, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAttachmentsByIncidentID(ctx, uuidToPgtype(incidentID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Attachment, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAttachment(row))
	}
	return out, nil
}

// CountByIncidentID implementa domain.AttachmentRepository.
func (r *AttachmentRepository) CountByIncidentID(ctx context.Context, incidentID string) (int, error) {
	q, err := querier(ctx)
	if err != nil {
		return 0, err
	}
	count, err := q.CountAttachmentsByIncidentID(ctx, uuidToPgtype(incidentID))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// --- StatusHistoryRepository ---

// StatusHistoryRepository implementa domain.StatusHistoryRepository.
type StatusHistoryRepository struct{}

// NewStatusHistoryRepository construye una instancia stateless.
func NewStatusHistoryRepository() *StatusHistoryRepository {
	return &StatusHistoryRepository{}
}

// Record implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) Record(ctx context.Context, in domain.RecordStatusHistoryInput) (entities.StatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.StatusHistory{}, err
	}
	row, err := q.RecordIncidentStatusHistory(ctx, incidentsdb.RecordIncidentStatusHistoryParams{
		IncidentID:           uuidToPgtype(in.IncidentID),
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

// ListByIncidentID implementa domain.StatusHistoryRepository.
func (r *StatusHistoryRepository) ListByIncidentID(ctx context.Context, incidentID string) ([]entities.StatusHistory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListStatusHistoryByIncidentID(ctx, uuidToPgtype(incidentID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.StatusHistory, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapStatusHistory(row))
	}
	return out, nil
}

// --- IncidentAssignmentRepository ---

// IncidentAssignmentRepository implementa domain.IncidentAssignmentRepository.
type IncidentAssignmentRepository struct{}

// NewIncidentAssignmentRepository construye una instancia stateless.
func NewIncidentAssignmentRepository() *IncidentAssignmentRepository {
	return &IncidentAssignmentRepository{}
}

// Create implementa domain.IncidentAssignmentRepository.
func (r *IncidentAssignmentRepository) Create(ctx context.Context, in domain.CreateAssignmentInput) (entities.IncidentAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.IncidentAssignment{}, err
	}
	row, err := q.CreateIncidentAssignment(ctx, incidentsdb.CreateIncidentAssignmentParams{
		IncidentID:       uuidToPgtype(in.IncidentID),
		AssignedToUserID: uuidToPgtype(in.AssignedToUserID),
		AssignedByUserID: uuidToPgtype(in.AssignedByUserID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.IncidentAssignment{}, domain.ErrAssignmentAlreadyActive
		}
		return entities.IncidentAssignment{}, err
	}
	return mapAssignment(row), nil
}

// UnassignActive implementa domain.IncidentAssignmentRepository.
func (r *IncidentAssignmentRepository) UnassignActive(ctx context.Context, incidentID string, actorID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.UnassignActiveByIncidentID(ctx, incidentsdb.UnassignActiveByIncidentIDParams{
		UpdatedBy:  uuidToPgtype(actorID),
		IncidentID: uuidToPgtype(incidentID),
	})
}

// GetActiveByIncidentID implementa domain.IncidentAssignmentRepository.
func (r *IncidentAssignmentRepository) GetActiveByIncidentID(ctx context.Context, incidentID string) (entities.IncidentAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.IncidentAssignment{}, err
	}
	row, err := q.GetActiveAssignmentByIncidentID(ctx, uuidToPgtype(incidentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.IncidentAssignment{}, domain.ErrAttachmentNotFound
		}
		return entities.IncidentAssignment{}, err
	}
	return mapAssignment(row), nil
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
	row, err := q.EnqueueIncidentOutboxEvent(ctx, incidentsdb.EnqueueIncidentOutboxEventParams{
		IncidentID: uuidToPgtype(in.IncidentID),
		EventType:  string(in.EventType),
		Payload:    in.Payload,
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
	rows, err := q.LockPendingIncidentOutboxEvents(ctx, limit)
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
	return q.MarkIncidentOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkIncidentOutboxEventFailed(ctx, incidentsdb.MarkIncidentOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// --- helpers de mapeo ---

func mapIncident(r incidentsdb.Incident) entities.Incident {
	out := entities.Incident{
		ID:               uuidString(r.ID),
		IncidentType:     entities.IncidentType(r.IncidentType),
		Severity:         entities.Severity(r.Severity),
		Title:            r.Title,
		Description:      r.Description,
		ReportedByUserID: uuidString(r.ReportedByUserID),
		ReportedAt:       tsToTime(r.ReportedAt),
		LocationDetail:   r.LocationDetail,
		ResolutionNotes:  r.ResolutionNotes,
		Escalated:        r.Escalated,
		Status:           entities.IncidentStatus(r.Status),
		CreatedAt:        tsToTime(r.CreatedAt),
		UpdatedAt:        tsToTime(r.UpdatedAt),
		Version:          r.Version,
	}
	if s := uuidStringPtr(r.StructureID); s != nil {
		out.StructureID = s
	}
	if s := uuidStringPtr(r.AssignedToUserID); s != nil {
		out.AssignedToUserID = s
	}
	if r.AssignedAt.Valid {
		t := r.AssignedAt.Time
		out.AssignedAt = &t
	}
	if r.StartedAt.Valid {
		t := r.StartedAt.Time
		out.StartedAt = &t
	}
	if r.ResolvedAt.Valid {
		t := r.ResolvedAt.Time
		out.ResolvedAt = &t
	}
	if r.ClosedAt.Valid {
		t := r.ClosedAt.Time
		out.ClosedAt = &t
	}
	if r.CancelledAt.Valid {
		t := r.CancelledAt.Time
		out.CancelledAt = &t
	}
	if r.SlaAssignDueAt.Valid {
		t := r.SlaAssignDueAt.Time
		out.SLAAssignDueAt = &t
	}
	if r.SlaResolveDueAt.Valid {
		t := r.SlaResolveDueAt.Time
		out.SLAResolveDueAt = &t
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

func mapAttachment(r incidentsdb.IncidentAttachment) entities.Attachment {
	out := entities.Attachment{
		ID:         uuidString(r.ID),
		IncidentID: uuidString(r.IncidentID),
		URL:        r.Url,
		MimeType:   r.MimeType,
		SizeBytes:  r.SizeBytes,
		UploadedBy: uuidString(r.UploadedBy),
		Status:     r.Status,
		CreatedAt:  tsToTime(r.CreatedAt),
		UpdatedAt:  tsToTime(r.UpdatedAt),
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

func mapStatusHistory(r incidentsdb.IncidentStatusHistory) entities.StatusHistory {
	out := entities.StatusHistory{
		ID:                   uuidString(r.ID),
		IncidentID:           uuidString(r.IncidentID),
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

func mapAssignment(r incidentsdb.IncidentAssignment) entities.IncidentAssignment {
	out := entities.IncidentAssignment{
		ID:               uuidString(r.ID),
		IncidentID:       uuidString(r.IncidentID),
		AssignedToUserID: uuidString(r.AssignedToUserID),
		AssignedByUserID: uuidString(r.AssignedByUserID),
		AssignedAt:       tsToTime(r.AssignedAt),
		Status:           entities.IncidentAssignmentStatus(r.Status),
		CreatedAt:        tsToTime(r.CreatedAt),
		UpdatedAt:        tsToTime(r.UpdatedAt),
	}
	if r.UnassignedAt.Valid {
		t := r.UnassignedAt.Time
		out.UnassignedAt = &t
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

func mapOutbox(r incidentsdb.IncidentOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:            uuidString(r.ID),
		IncidentID:    uuidString(r.IncidentID),
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
	_ domain.IncidentRepository           = (*IncidentRepository)(nil)
	_ domain.AttachmentRepository         = (*AttachmentRepository)(nil)
	_ domain.StatusHistoryRepository      = (*StatusHistoryRepository)(nil)
	_ domain.IncidentAssignmentRepository = (*IncidentAssignmentRepository)(nil)
	_ domain.OutboxRepository             = (*OutboxRepository)(nil)
)
