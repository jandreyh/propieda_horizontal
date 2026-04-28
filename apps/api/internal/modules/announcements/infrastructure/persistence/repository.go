// Package persistence implementa los puertos del modulo announcements
// usando el codigo generado por sqlc.
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

	"github.com/saas-ph/api/internal/modules/announcements/domain"
	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
	announcementsdb "github.com/saas-ph/api/internal/modules/announcements/infrastructure/persistence/sqlcgen"
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

func querier(ctx context.Context) (*announcementsdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return announcementsdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("announcements: tenant pool is nil")
	}
	return announcementsdb.New(t.Pool), nil
}

// --- AnnouncementRepository ---

// AnnouncementRepository implementa domain.AnnouncementRepository.
type AnnouncementRepository struct{}

// NewAnnouncementRepository construye una instancia stateless.
func NewAnnouncementRepository() *AnnouncementRepository { return &AnnouncementRepository{} }

// Create implementa domain.AnnouncementRepository.
func (r *AnnouncementRepository) Create(ctx context.Context, in domain.CreateAnnouncementInput) (entities.Announcement, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Announcement{}, err
	}
	row, err := q.CreateAnnouncement(ctx, announcementsdb.CreateAnnouncementParams{
		Title:             in.Title,
		Body:              in.Body,
		PublishedByUserID: uuidToPgtype(in.PublishedByUserID),
		Pinned:            in.Pinned,
		ExpiresAt:         tsFromPtr(in.ExpiresAt),
	})
	if err != nil {
		return entities.Announcement{}, err
	}
	return mapAnnouncement(row), nil
}

// GetByID implementa domain.AnnouncementRepository.
func (r *AnnouncementRepository) GetByID(ctx context.Context, id string) (entities.Announcement, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Announcement{}, err
	}
	row, err := q.GetAnnouncementByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Announcement{}, domain.ErrAnnouncementNotFound
		}
		return entities.Announcement{}, err
	}
	return mapAnnouncement(row), nil
}

// Archive implementa domain.AnnouncementRepository.
func (r *AnnouncementRepository) Archive(ctx context.Context, id, actorID string) (entities.Announcement, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Announcement{}, err
	}
	row, err := q.ArchiveAnnouncement(ctx, announcementsdb.ArchiveAnnouncementParams{
		DeletedBy: uuidToPgtype(actorID),
		ID:        uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Announcement{}, domain.ErrAnnouncementNotFound
		}
		return entities.Announcement{}, err
	}
	return mapAnnouncement(row), nil
}

// ListFeedForUser implementa domain.AnnouncementRepository.
func (r *AnnouncementRepository) ListFeedForUser(ctx context.Context, fq domain.FeedQuery) ([]entities.Announcement, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListFeedForUser(ctx, announcementsdb.ListFeedForUserParams{
		UserID:       uuidToPgtype(fq.UserID),
		RoleIds:      uuidsToPgtype(fq.RoleIDs),
		StructureIds: uuidsToPgtype(fq.StructureIDs),
		UnitIds:      uuidsToPgtype(fq.UnitIDs),
		Off:          fq.Offset,
		Lim:          fq.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]entities.Announcement, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAnnouncement(row))
	}
	return out, nil
}

// --- AudienceRepository ---

// AudienceRepository implementa domain.AudienceRepository.
type AudienceRepository struct{}

// NewAudienceRepository construye una instancia stateless.
func NewAudienceRepository() *AudienceRepository { return &AudienceRepository{} }

// Add implementa domain.AudienceRepository.
func (r *AudienceRepository) Add(ctx context.Context, in domain.AddAudienceInput) (entities.Audience, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Audience{}, err
	}
	var targetID pgtype.UUID
	if in.TargetID != nil {
		targetID = uuidToPgtype(*in.TargetID)
	}
	row, err := q.AddAudience(ctx, announcementsdb.AddAudienceParams{
		AnnouncementID: uuidToPgtype(in.AnnouncementID),
		TargetType:     string(in.TargetType),
		TargetID:       targetID,
		CreatedBy:      uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.Audience{}, err
	}
	return mapAudience(row), nil
}

// ListByAnnouncement implementa domain.AudienceRepository.
func (r *AudienceRepository) ListByAnnouncement(ctx context.Context, announcementID string) ([]entities.Audience, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAudiencesByAnnouncement(ctx, uuidToPgtype(announcementID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Audience, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAudience(row))
	}
	return out, nil
}

// --- AckRepository ---

// AckRepository implementa domain.AckRepository.
type AckRepository struct{}

// NewAckRepository construye una instancia stateless.
func NewAckRepository() *AckRepository { return &AckRepository{} }

// Acknowledge implementa domain.AckRepository.
func (r *AckRepository) Acknowledge(ctx context.Context, announcementID, userID string) (entities.Ack, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Ack{}, err
	}
	row, err := q.Acknowledge(ctx, announcementsdb.AcknowledgeParams{
		AnnouncementID: uuidToPgtype(announcementID),
		UserID:         uuidToPgtype(userID),
	})
	if err != nil {
		return entities.Ack{}, err
	}
	return entities.Ack{
		ID:             uuidString(row.ID),
		AnnouncementID: uuidString(row.AnnouncementID),
		UserID:         uuidString(row.UserID),
		AcknowledgedAt: tsToTime(row.AcknowledgedAt),
		CreatedAt:      tsToTime(row.CreatedAt),
		UpdatedAt:      tsToTime(row.UpdatedAt),
		CreatedBy:      uuidStringPtr(row.CreatedBy),
		UpdatedBy:      uuidStringPtr(row.UpdatedBy),
	}, nil
}

// --- helpers de mapeo ---

func mapAnnouncement(r announcementsdb.Announcement) entities.Announcement {
	out := entities.Announcement{
		ID:                uuidString(r.ID),
		Title:             r.Title,
		Body:              r.Body,
		PublishedByUserID: uuidString(r.PublishedByUserID),
		PublishedAt:       tsToTime(r.PublishedAt),
		Pinned:            r.Pinned,
		Status:            entities.AnnouncementStatus(r.Status),
		CreatedAt:         tsToTime(r.CreatedAt),
		UpdatedAt:         tsToTime(r.UpdatedAt),
		Version:           r.Version,
	}
	if r.ExpiresAt.Valid {
		t := r.ExpiresAt.Time
		out.ExpiresAt = &t
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

func mapAudience(r announcementsdb.AnnouncementAudience) entities.Audience {
	out := entities.Audience{
		ID:             uuidString(r.ID),
		AnnouncementID: uuidString(r.AnnouncementID),
		TargetType:     entities.TargetType(r.TargetType),
		Status:         r.Status,
		CreatedAt:      tsToTime(r.CreatedAt),
		UpdatedAt:      tsToTime(r.UpdatedAt),
	}
	if s := uuidStringPtr(r.TargetID); s != nil {
		out.TargetID = s
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

// --- pgtype helpers ---

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func tsFromPtr(t *time.Time) pgtype.Timestamptz {
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

// uuidsToPgtype convierte un slice de strings a un slice de pgtype.UUID
// (validos). Los strings invalidos se omiten silenciosamente — la capa
// de aplicacion ya validó. Devuelve siempre slice no-nil para que pgx lo
// codifique como `'{}'::uuid[]` cuando no hay elementos.
func uuidsToPgtype(ss []string) []pgtype.UUID {
	out := make([]pgtype.UUID, 0, len(ss))
	for _, s := range ss {
		if s == "" {
			continue
		}
		var u pgtype.UUID
		if err := u.Scan(s); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// Compile-time checks: el repo implementa el puerto del dominio.
var (
	_ domain.AnnouncementRepository = (*AnnouncementRepository)(nil)
	_ domain.AudienceRepository     = (*AudienceRepository)(nil)
	_ domain.AckRepository          = (*AckRepository)(nil)
)
