// Package persistence implementa los puertos del modulo packages usando
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

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	packagesdb "github.com/saas-ph/api/internal/modules/packages/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// --- ctx helper para transaccion ---

type txCtxKey struct{}

// WithTx inyecta una transaccion pgx en el contexto. Cuando los repos
// resuelvan su `Querier`, prefieren la tx si esta presente.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// txFromCtx extrae una tx pgx del contexto si existe.
func txFromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	return tx, ok
}

func querier(ctx context.Context) (*packagesdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return packagesdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("packages: tenant pool is nil")
	}
	return packagesdb.New(t.Pool), nil
}

// --- CategoryRepository ---

// CategoryRepository implementa domain.CategoryRepository.
type CategoryRepository struct{}

// NewCategoryRepository construye una instancia stateless.
func NewCategoryRepository() *CategoryRepository { return &CategoryRepository{} }

// List implementa domain.CategoryRepository.
func (r *CategoryRepository) List(ctx context.Context) ([]entities.PackageCategory, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.PackageCategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCategory(row))
	}
	return out, nil
}

// GetByName implementa domain.CategoryRepository.
func (r *CategoryRepository) GetByName(ctx context.Context, name string) (entities.PackageCategory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PackageCategory{}, err
	}
	row, err := q.GetCategoryByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PackageCategory{}, domain.ErrCategoryNotFound
		}
		return entities.PackageCategory{}, err
	}
	return mapCategory(row), nil
}

// GetByID implementa domain.CategoryRepository.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (entities.PackageCategory, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PackageCategory{}, err
	}
	row, err := q.GetCategoryByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PackageCategory{}, domain.ErrCategoryNotFound
		}
		return entities.PackageCategory{}, err
	}
	return mapCategory(row), nil
}

// --- PackageRepository ---

// PackageRepository implementa domain.PackageRepository.
type PackageRepository struct{}

// NewPackageRepository construye una instancia stateless.
func NewPackageRepository() *PackageRepository { return &PackageRepository{} }

// Create implementa domain.PackageRepository.
func (r *PackageRepository) Create(ctx context.Context, in domain.CreatePackageInput) (entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Package{}, err
	}
	row, err := q.CreatePackage(ctx, packagesdb.CreatePackageParams{
		UnitID:              uuidToPgtype(in.UnitID),
		RecipientName:       in.RecipientName,
		CategoryID:          uuidToPgtypePtr(in.CategoryID),
		ReceivedEvidenceUrl: in.ReceivedEvidenceURL,
		Carrier:             in.Carrier,
		TrackingNumber:      in.TrackingNumber,
		ReceivedByUserID:    uuidToPgtype(in.ReceivedByUserID),
	})
	if err != nil {
		return entities.Package{}, err
	}
	return mapPackage(row), nil
}

// GetByID implementa domain.PackageRepository.
func (r *PackageRepository) GetByID(ctx context.Context, id string) (entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Package{}, err
	}
	row, err := q.GetPackageByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Package{}, domain.ErrPackageNotFound
		}
		return entities.Package{}, err
	}
	return mapPackage(row), nil
}

// ListByUnit implementa domain.PackageRepository.
func (r *PackageRepository) ListByUnit(ctx context.Context, unitID string) ([]entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPackagesByUnit(ctx, uuidToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Package, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPackage(row))
	}
	return out, nil
}

// ListByStatus implementa domain.PackageRepository.
func (r *PackageRepository) ListByStatus(ctx context.Context, status entities.PackageStatus) ([]entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPackagesByStatus(ctx, string(status))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Package, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPackage(row))
	}
	return out, nil
}

// UpdateStatusOptimistic implementa domain.PackageRepository.
func (r *PackageRepository) UpdateStatusOptimistic(
	ctx context.Context,
	id string,
	expectedVersion int32,
	newStatus entities.PackageStatus,
	actorID string,
) (entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Package{}, err
	}
	row, err := q.UpdatePackageStatus(ctx, packagesdb.UpdatePackageStatusParams{
		NewStatus:       string(newStatus),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Package{}, domain.ErrVersionConflict
		}
		return entities.Package{}, err
	}
	return mapPackage(row), nil
}

// Return implementa domain.PackageRepository.
func (r *PackageRepository) Return(
	ctx context.Context,
	id string,
	expectedVersion int32,
	actorID string,
) (entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Package{}, err
	}
	row, err := q.ReturnPackage(ctx, packagesdb.ReturnPackageParams{
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Package{}, domain.ErrVersionConflict
		}
		return entities.Package{}, err
	}
	return mapPackage(row), nil
}

// ListPendingReminder implementa domain.PackageRepository.
func (r *PackageRepository) ListPendingReminder(ctx context.Context) ([]entities.Package, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPackagesPendingReminder(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Package, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPackage(row))
	}
	return out, nil
}

// --- DeliveryRepository ---

// DeliveryRepository implementa domain.DeliveryRepository.
type DeliveryRepository struct{}

// NewDeliveryRepository construye una instancia stateless.
func NewDeliveryRepository() *DeliveryRepository { return &DeliveryRepository{} }

// Record implementa domain.DeliveryRepository.
func (r *DeliveryRepository) Record(ctx context.Context, in domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.DeliveryEvent{}, err
	}
	row, err := q.RecordDeliveryEvent(ctx, packagesdb.RecordDeliveryEventParams{
		PackageID:           uuidToPgtype(in.PackageID),
		DeliveredToUserID:   uuidToPgtypePtr(in.DeliveredToUserID),
		RecipientNameManual: in.RecipientNameManual,
		DeliveryMethod:      string(in.DeliveryMethod),
		SignatureUrl:        in.SignatureURL,
		PhotoEvidenceUrl:    in.PhotoEvidenceURL,
		DeliveredByUserID:   uuidToPgtype(in.DeliveredByUserID),
		Notes:               in.Notes,
	})
	if err != nil {
		return entities.DeliveryEvent{}, err
	}
	return mapDeliveryEvent(row), nil
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
	row, err := q.EnqueueOutboxEvent(ctx, packagesdb.EnqueueOutboxEventParams{
		PackageID: uuidToPgtype(in.PackageID),
		EventType: string(in.EventType),
		Payload:   in.Payload,
	})
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	return mapOutbox(row), nil
}

// LockPending implementa domain.OutboxRepository. DEBE invocarse dentro
// de una transaccion (pasada via WithTx en el contexto).
func (r *OutboxRepository) LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.LockPendingOutboxEvents(ctx, limit)
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
	return q.MarkOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkOutboxEventFailed(ctx, packagesdb.MarkOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// --- helpers de mapeo ---

func mapCategory(r packagesdb.PackageCategory) entities.PackageCategory {
	out := entities.PackageCategory{
		ID:               uuidString(r.ID),
		Name:             r.Name,
		RequiresEvidence: r.RequiresEvidence,
		Status:           r.Status,
		CreatedAt:        tsToTime(r.CreatedAt),
		UpdatedAt:        tsToTime(r.UpdatedAt),
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

func mapPackage(r packagesdb.Package) entities.Package {
	out := entities.Package{
		ID:                  uuidString(r.ID),
		UnitID:              uuidString(r.UnitID),
		RecipientName:       r.RecipientName,
		ReceivedEvidenceURL: r.ReceivedEvidenceUrl,
		Carrier:             r.Carrier,
		TrackingNumber:      r.TrackingNumber,
		ReceivedByUserID:    uuidString(r.ReceivedByUserID),
		ReceivedAt:          tsToTime(r.ReceivedAt),
		Status:              entities.PackageStatus(r.Status),
		CreatedAt:           tsToTime(r.CreatedAt),
		UpdatedAt:           tsToTime(r.UpdatedAt),
		Version:             r.Version,
	}
	if s := uuidStringPtr(r.CategoryID); s != nil {
		out.CategoryID = s
	}
	if r.DeliveredAt.Valid {
		t := r.DeliveredAt.Time
		out.DeliveredAt = &t
	}
	if r.ReturnedAt.Valid {
		t := r.ReturnedAt.Time
		out.ReturnedAt = &t
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

func mapDeliveryEvent(r packagesdb.PackageDeliveryEvent) entities.DeliveryEvent {
	out := entities.DeliveryEvent{
		ID:                  uuidString(r.ID),
		PackageID:           uuidString(r.PackageID),
		RecipientNameManual: r.RecipientNameManual,
		DeliveryMethod:      entities.DeliveryMethod(r.DeliveryMethod),
		SignatureURL:        r.SignatureUrl,
		PhotoEvidenceURL:    r.PhotoEvidenceUrl,
		DeliveredByUserID:   uuidString(r.DeliveredByUserID),
		DeliveredAt:         tsToTime(r.DeliveredAt),
		Notes:               r.Notes,
		Status:              entities.DeliveryEventStatus(r.Status),
		CreatedAt:           tsToTime(r.CreatedAt),
		UpdatedAt:           tsToTime(r.UpdatedAt),
		Version:             r.Version,
	}
	if s := uuidStringPtr(r.DeliveredToUserID); s != nil {
		out.DeliveredToUserID = s
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

func mapOutbox(r packagesdb.PackageOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:            uuidString(r.ID),
		PackageID:     uuidString(r.PackageID),
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

// Compile-time checks: el repo implementa el puerto del dominio.
var (
	_ domain.CategoryRepository = (*CategoryRepository)(nil)
	_ domain.PackageRepository  = (*PackageRepository)(nil)
	_ domain.DeliveryRepository = (*DeliveryRepository)(nil)
	_ domain.OutboxRepository   = (*OutboxRepository)(nil)
)
