// Package persistence implementa los puertos del modulo notifications
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

	"github.com/saas-ph/api/internal/modules/notifications/domain"
	"github.com/saas-ph/api/internal/modules/notifications/domain/entities"
	ndb "github.com/saas-ph/api/internal/modules/notifications/infrastructure/persistence/sqlcgen"
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

func querier(ctx context.Context) (*ndb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return ndb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("notifications: tenant pool is nil")
	}
	return ndb.New(t.Pool), nil
}

// --- TemplateRepository ---

// TemplateRepository implementa domain.TemplateRepository.
type TemplateRepository struct{}

// NewTemplateRepository construye una instancia stateless.
func NewTemplateRepository() *TemplateRepository { return &TemplateRepository{} }

// Create implementa domain.TemplateRepository.
func (r *TemplateRepository) Create(ctx context.Context, in domain.CreateTemplateInput) (entities.NotificationTemplate, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationTemplate{}, err
	}
	row, err := q.CreateNotificationTemplate(ctx, ndb.CreateNotificationTemplateParams{
		EventType:           in.EventType,
		Channel:             string(in.Channel),
		Locale:              in.Locale,
		Subject:             in.Subject,
		BodyTemplate:        in.BodyTemplate,
		ProviderTemplateRef: in.ProviderTemplateRef,
		CreatedBy:           uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.NotificationTemplate{}, domain.ErrTemplateDuplicate
		}
		return entities.NotificationTemplate{}, err
	}
	return mapTemplate(row), nil
}

// GetByID implementa domain.TemplateRepository.
func (r *TemplateRepository) GetByID(ctx context.Context, id string) (entities.NotificationTemplate, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationTemplate{}, err
	}
	row, err := q.GetNotificationTemplateByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationTemplate{}, domain.ErrTemplateNotFound
		}
		return entities.NotificationTemplate{}, err
	}
	return mapTemplate(row), nil
}

// List implementa domain.TemplateRepository.
func (r *TemplateRepository) List(ctx context.Context) ([]entities.NotificationTemplate, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListNotificationTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationTemplate, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTemplate(row))
	}
	return out, nil
}

// Update implementa domain.TemplateRepository.
func (r *TemplateRepository) Update(ctx context.Context, in domain.UpdateTemplateInput) (entities.NotificationTemplate, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationTemplate{}, err
	}
	row, err := q.UpdateNotificationTemplate(ctx, ndb.UpdateNotificationTemplateParams{
		NewSubject:             in.Subject,
		NewBodyTemplate:        in.BodyTemplate,
		NewProviderTemplateRef: in.ProviderTemplateRef,
		UpdatedBy:              uuidToPgtype(in.ActorID),
		ID:                     uuidToPgtype(in.ID),
		ExpectedVersion:        in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationTemplate{}, domain.ErrVersionConflict
		}
		return entities.NotificationTemplate{}, err
	}
	return mapTemplate(row), nil
}

// --- PreferenceRepository ---

// PreferenceRepository implementa domain.PreferenceRepository.
type PreferenceRepository struct{}

// NewPreferenceRepository construye una instancia stateless.
func NewPreferenceRepository() *PreferenceRepository { return &PreferenceRepository{} }

// Upsert implementa domain.PreferenceRepository.
func (r *PreferenceRepository) Upsert(ctx context.Context, in domain.UpsertPreferenceInput) (entities.NotificationPreference, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationPreference{}, err
	}
	row, err := q.UpsertNotificationPreference(ctx, ndb.UpsertNotificationPreferenceParams{
		UserID:    uuidToPgtype(in.UserID),
		EventType: in.EventType,
		Channel:   string(in.Channel),
		Enabled:   in.Enabled,
		CreatedBy: uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.NotificationPreference{}, err
	}
	return mapPreference(row), nil
}

// ListByUserID implementa domain.PreferenceRepository.
func (r *PreferenceRepository) ListByUserID(ctx context.Context, userID string) ([]entities.NotificationPreference, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListNotificationPreferencesByUserID(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationPreference, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPreference(row))
	}
	return out, nil
}

// GetByUserEventChannel implementa domain.PreferenceRepository.
func (r *PreferenceRepository) GetByUserEventChannel(ctx context.Context, userID, eventType string, channel entities.Channel) (entities.NotificationPreference, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationPreference{}, err
	}
	row, err := q.GetNotificationPreferenceByUserEventChannel(ctx, uuidToPgtype(userID), eventType, string(channel))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationPreference{}, domain.ErrPreferenceNotFound
		}
		return entities.NotificationPreference{}, err
	}
	return mapPreference(row), nil
}

// --- ConsentRepository ---

// ConsentRepository implementa domain.ConsentRepository.
type ConsentRepository struct{}

// NewConsentRepository construye una instancia stateless.
func NewConsentRepository() *ConsentRepository { return &ConsentRepository{} }

// Create implementa domain.ConsentRepository.
func (r *ConsentRepository) Create(ctx context.Context, in domain.CreateConsentInput) (entities.NotificationConsent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationConsent{}, err
	}
	row, err := q.CreateNotificationConsent(ctx, ndb.CreateNotificationConsentParams{
		UserID:          uuidToPgtype(in.UserID),
		Channel:         string(in.Channel),
		ConsentProofURL: in.ConsentProofURL,
		LegalBasis:      in.LegalBasis,
		CreatedBy:       uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.NotificationConsent{}, domain.ErrConsentDuplicate
		}
		return entities.NotificationConsent{}, err
	}
	return mapConsent(row), nil
}

// GetActiveByUserChannel implementa domain.ConsentRepository.
func (r *ConsentRepository) GetActiveByUserChannel(ctx context.Context, userID string, channel entities.Channel) (entities.NotificationConsent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationConsent{}, err
	}
	row, err := q.GetActiveNotificationConsentByUserChannel(ctx, uuidToPgtype(userID), string(channel))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationConsent{}, domain.ErrConsentNotFound
		}
		return entities.NotificationConsent{}, err
	}
	return mapConsent(row), nil
}

// Revoke implementa domain.ConsentRepository.
func (r *ConsentRepository) Revoke(ctx context.Context, userID string, channel entities.Channel, actorID string) (entities.NotificationConsent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationConsent{}, err
	}
	row, err := q.RevokeNotificationConsent(ctx, ndb.RevokeNotificationConsentParams{
		UpdatedBy: uuidToPgtype(actorID),
		UserID:    uuidToPgtype(userID),
		Channel:   string(channel),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationConsent{}, domain.ErrConsentNotFound
		}
		return entities.NotificationConsent{}, err
	}
	return mapConsent(row), nil
}

// ListByUserID implementa domain.ConsentRepository.
func (r *ConsentRepository) ListByUserID(ctx context.Context, userID string) ([]entities.NotificationConsent, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListNotificationConsentsByUserID(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationConsent, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapConsent(row))
	}
	return out, nil
}

// --- PushTokenRepository ---

// PushTokenRepository implementa domain.PushTokenRepository.
type PushTokenRepository struct{}

// NewPushTokenRepository construye una instancia stateless.
func NewPushTokenRepository() *PushTokenRepository { return &PushTokenRepository{} }

// Create implementa domain.PushTokenRepository.
func (r *PushTokenRepository) Create(ctx context.Context, in domain.CreatePushTokenInput) (entities.NotificationPushToken, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationPushToken{}, err
	}
	row, err := q.CreateNotificationPushToken(ctx, ndb.CreateNotificationPushTokenParams{
		UserID:    uuidToPgtype(in.UserID),
		Platform:  string(in.Platform),
		Token:     in.Token,
		CreatedBy: uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.NotificationPushToken{}, domain.ErrPushTokenDuplicate
		}
		return entities.NotificationPushToken{}, err
	}
	return mapPushToken(row), nil
}

// GetByID implementa domain.PushTokenRepository.
func (r *PushTokenRepository) GetByID(ctx context.Context, id string) (entities.NotificationPushToken, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationPushToken{}, err
	}
	row, err := q.GetNotificationPushTokenByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationPushToken{}, domain.ErrPushTokenNotFound
		}
		return entities.NotificationPushToken{}, err
	}
	return mapPushToken(row), nil
}

// ListActiveByUserID implementa domain.PushTokenRepository.
func (r *PushTokenRepository) ListActiveByUserID(ctx context.Context, userID string) ([]entities.NotificationPushToken, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveNotificationPushTokensByUserID(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationPushToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPushToken(row))
	}
	return out, nil
}

// SoftDelete implementa domain.PushTokenRepository.
func (r *PushTokenRepository) SoftDelete(ctx context.Context, id string, actorID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.SoftDeleteNotificationPushToken(ctx, ndb.SoftDeleteNotificationPushTokenParams{
		DeletedBy: uuidToPgtype(actorID),
		ID:        uuidToPgtype(id),
	})
}

// TouchLastSeen implementa domain.PushTokenRepository.
func (r *PushTokenRepository) TouchLastSeen(ctx context.Context, id string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.TouchNotificationPushTokenLastSeen(ctx, uuidToPgtype(id))
}

// --- ProviderConfigRepository ---

// ProviderConfigRepository implementa domain.ProviderConfigRepository.
type ProviderConfigRepository struct{}

// NewProviderConfigRepository construye una instancia stateless.
func NewProviderConfigRepository() *ProviderConfigRepository { return &ProviderConfigRepository{} }

// List implementa domain.ProviderConfigRepository.
func (r *ProviderConfigRepository) List(ctx context.Context) ([]entities.NotificationProviderConfig, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListNotificationProviderConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationProviderConfig, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapProviderConfig(row))
	}
	return out, nil
}

// GetActiveByChannel implementa domain.ProviderConfigRepository.
func (r *ProviderConfigRepository) GetActiveByChannel(ctx context.Context, channel entities.Channel) (entities.NotificationProviderConfig, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationProviderConfig{}, err
	}
	row, err := q.GetActiveNotificationProviderConfigByChannel(ctx, string(channel))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationProviderConfig{}, domain.ErrProviderConfigNotFound
		}
		return entities.NotificationProviderConfig{}, err
	}
	return mapProviderConfig(row), nil
}

// Update implementa domain.ProviderConfigRepository.
func (r *ProviderConfigRepository) Update(ctx context.Context, in domain.UpdateProviderConfigInput) (entities.NotificationProviderConfig, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationProviderConfig{}, err
	}
	row, err := q.UpdateNotificationProviderConfig(ctx, ndb.UpdateNotificationProviderConfigParams{
		NewConfig:       in.Config,
		NewIsActive:     in.IsActive,
		UpdatedBy:       uuidToPgtype(in.ActorID),
		ID:              uuidToPgtype(in.ID),
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationProviderConfig{}, domain.ErrVersionConflict
		}
		return entities.NotificationProviderConfig{}, err
	}
	return mapProviderConfig(row), nil
}

// --- OutboxRepository ---

// OutboxRepository implementa domain.OutboxRepository.
type OutboxRepository struct{}

// NewOutboxRepository construye una instancia stateless.
func NewOutboxRepository() *OutboxRepository { return &OutboxRepository{} }

// Enqueue implementa domain.OutboxRepository.
func (r *OutboxRepository) Enqueue(ctx context.Context, in domain.EnqueueOutboxInput) (entities.NotificationOutbox, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationOutbox{}, err
	}
	row, err := q.EnqueueNotificationOutbox(ctx, ndb.EnqueueNotificationOutboxParams{
		EventType:       in.EventType,
		RecipientUserID: uuidToPgtype(in.RecipientUserID),
		Channel:         string(in.Channel),
		Payload:         in.Payload,
		IdempotencyKey:  in.IdempotencyKey,
		ScheduledAt:     timeToPgTimestamptz(in.ScheduledAt),
		CreatedBy:       uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.NotificationOutbox{}, domain.ErrOutboxDuplicate
		}
		return entities.NotificationOutbox{}, err
	}
	return mapOutbox(row), nil
}

// GetByID implementa domain.OutboxRepository.
func (r *OutboxRepository) GetByID(ctx context.Context, id string) (entities.NotificationOutbox, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationOutbox{}, err
	}
	row, err := q.GetNotificationOutboxByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.NotificationOutbox{}, domain.ErrOutboxNotFound
		}
		return entities.NotificationOutbox{}, err
	}
	return mapOutbox(row), nil
}

// ListPending implementa domain.OutboxRepository.
func (r *OutboxRepository) ListPending(ctx context.Context, limit int32) ([]entities.NotificationOutbox, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPendingNotificationOutbox(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationOutbox, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOutbox(row))
	}
	return out, nil
}

// MarkSending implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkSending(ctx context.Context, id string, expectedVersion int32) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.MarkNotificationOutboxSending(ctx, ndb.MarkNotificationOutboxSendingParams{
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// MarkSent implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkSent(ctx context.Context, id string, expectedVersion int32) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.MarkNotificationOutboxSent(ctx, ndb.MarkNotificationOutboxSentParams{
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// MarkFailedRetry implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailedRetry(ctx context.Context, id string, expectedVersion int32, lastError string, nextAttemptAt time.Time) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	le := lastError
	return q.MarkNotificationOutboxFailedRetry(ctx, ndb.MarkNotificationOutboxFailedRetryParams{
		LastError:       &le,
		NextAttemptAt:   pgtype.Timestamptz{Time: nextAttemptAt, Valid: true},
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// MarkFailedPermanent implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailedPermanent(ctx context.Context, id string, expectedVersion int32, lastError string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	le := lastError
	return q.MarkNotificationOutboxFailedPermanent(ctx, ndb.MarkNotificationOutboxFailedPermanentParams{
		LastError:       &le,
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// MarkBlockedNoConsent implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkBlockedNoConsent(ctx context.Context, id string, expectedVersion int32) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.MarkNotificationOutboxBlockedNoConsent(ctx, ndb.MarkNotificationOutboxBlockedNoConsentParams{
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// ListByRecipient implementa domain.OutboxRepository.
func (r *OutboxRepository) ListByRecipient(ctx context.Context, recipientUserID string) ([]entities.NotificationOutbox, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListNotificationOutboxByRecipient(ctx, uuidToPgtype(recipientUserID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationOutbox, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOutbox(row))
	}
	return out, nil
}

// --- DeliveryRepository ---

// DeliveryRepository implementa domain.DeliveryRepository.
type DeliveryRepository struct{}

// NewDeliveryRepository construye una instancia stateless.
func NewDeliveryRepository() *DeliveryRepository { return &DeliveryRepository{} }

// Create implementa domain.DeliveryRepository.
func (r *DeliveryRepository) Create(ctx context.Context, in domain.CreateDeliveryInput) (entities.NotificationDelivery, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.NotificationDelivery{}, err
	}
	deliveredAt := pgtype.Timestamptz{Valid: false}
	if in.DeliveredAt != nil {
		deliveredAt = pgtype.Timestamptz{Time: *in.DeliveredAt, Valid: true}
	}
	row, err := q.CreateNotificationDelivery(ctx, ndb.CreateNotificationDeliveryParams{
		OutboxID:          uuidToPgtype(in.OutboxID),
		ProviderName:      in.ProviderName,
		ProviderMessageID: in.ProviderMessageID,
		ProviderStatus:    in.ProviderStatus,
		DeliveredAt:       deliveredAt,
		FailureReason:     in.FailureReason,
		CreatedBy:         uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.NotificationDelivery{}, err
	}
	return mapDelivery(row), nil
}

// ListByOutboxID implementa domain.DeliveryRepository.
func (r *DeliveryRepository) ListByOutboxID(ctx context.Context, outboxID string) ([]entities.NotificationDelivery, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListNotificationDeliveriesByOutboxID(ctx, uuidToPgtype(outboxID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.NotificationDelivery, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapDelivery(row))
	}
	return out, nil
}

// --- helpers de mapeo ---

func mapTemplate(r ndb.NotificationTemplate) entities.NotificationTemplate {
	out := entities.NotificationTemplate{
		ID:                  uuidString(r.ID),
		EventType:           r.EventType,
		Channel:             entities.Channel(r.Channel),
		Locale:              r.Locale,
		Subject:             r.Subject,
		BodyTemplate:        r.BodyTemplate,
		ProviderTemplateRef: r.ProviderTemplateRef,
		Status:              entities.TemplateStatus(r.Status),
		CreatedAt:           tsToTime(r.CreatedAt),
		UpdatedAt:           tsToTime(r.UpdatedAt),
		Version:             r.Version,
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

func mapPreference(r ndb.NotificationPreference) entities.NotificationPreference {
	out := entities.NotificationPreference{
		ID:        uuidString(r.ID),
		UserID:    uuidString(r.UserID),
		EventType: r.EventType,
		Channel:   entities.Channel(r.Channel),
		Enabled:   r.Enabled,
		Status:    entities.PreferenceStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
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

func mapConsent(r ndb.NotificationConsent) entities.NotificationConsent {
	out := entities.NotificationConsent{
		ID:              uuidString(r.ID),
		UserID:          uuidString(r.UserID),
		Channel:         entities.Channel(r.Channel),
		ConsentedAt:     tsToTime(r.ConsentedAt),
		ConsentProofURL: r.ConsentProofURL,
		LegalBasis:      r.LegalBasis,
		Status:          entities.ConsentStatus(r.Status),
		CreatedAt:       tsToTime(r.CreatedAt),
		UpdatedAt:       tsToTime(r.UpdatedAt),
		Version:         r.Version,
	}
	if r.RevokedAt.Valid {
		t := r.RevokedAt.Time
		out.RevokedAt = &t
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

func mapPushToken(r ndb.NotificationPushToken) entities.NotificationPushToken {
	out := entities.NotificationPushToken{
		ID:         uuidString(r.ID),
		UserID:     uuidString(r.UserID),
		Platform:   entities.Platform(r.Platform),
		Token:      r.Token,
		LastSeenAt: tsToTime(r.LastSeenAt),
		Status:     entities.PushTokenStatus(r.Status),
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

func mapProviderConfig(r ndb.NotificationProviderConfig) entities.NotificationProviderConfig {
	out := entities.NotificationProviderConfig{
		ID:           uuidString(r.ID),
		Channel:      entities.Channel(r.Channel),
		ProviderName: r.ProviderName,
		Config:       r.Config,
		IsActive:     r.IsActive,
		Status:       r.Status,
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

func mapOutbox(r ndb.NotificationOutbox) entities.NotificationOutbox {
	out := entities.NotificationOutbox{
		ID:              uuidString(r.ID),
		EventType:       r.EventType,
		RecipientUserID: uuidString(r.RecipientUserID),
		Channel:         entities.Channel(r.Channel),
		Payload:         r.Payload,
		IdempotencyKey:  r.IdempotencyKey,
		ScheduledAt:     tsToTime(r.ScheduledAt),
		Attempts:        r.Attempts,
		LastError:       r.LastError,
		Status:          entities.OutboxStatus(r.Status),
		CreatedAt:       tsToTime(r.CreatedAt),
		UpdatedAt:       tsToTime(r.UpdatedAt),
		Version:         r.Version,
	}
	if r.SentAt.Valid {
		t := r.SentAt.Time
		out.SentAt = &t
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

func mapDelivery(r ndb.NotificationDelivery) entities.NotificationDelivery {
	out := entities.NotificationDelivery{
		ID:                uuidString(r.ID),
		OutboxID:          uuidString(r.OutboxID),
		ProviderName:      r.ProviderName,
		ProviderMessageID: r.ProviderMessageID,
		ProviderStatus:    r.ProviderStatus,
		FailureReason:     r.FailureReason,
		Status:            entities.DeliveryStatus(r.Status),
		CreatedAt:         tsToTime(r.CreatedAt),
		UpdatedAt:         tsToTime(r.UpdatedAt),
	}
	if r.DeliveredAt.Valid {
		t := r.DeliveredAt.Time
		out.DeliveredAt = &t
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
	_ domain.TemplateRepository       = (*TemplateRepository)(nil)
	_ domain.PreferenceRepository     = (*PreferenceRepository)(nil)
	_ domain.ConsentRepository        = (*ConsentRepository)(nil)
	_ domain.PushTokenRepository      = (*PushTokenRepository)(nil)
	_ domain.ProviderConfigRepository = (*ProviderConfigRepository)(nil)
	_ domain.OutboxRepository         = (*OutboxRepository)(nil)
	_ domain.DeliveryRepository       = (*DeliveryRepository)(nil)
)
