// Package domain define los puertos del modulo notifications.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/notifications/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrTemplateNotFound se devuelve cuando una plantilla por id no existe.
var ErrTemplateNotFound = errors.New("notifications: template not found")

// ErrTemplateDuplicate se devuelve cuando se intenta crear una plantilla
// con (event_type, channel, locale) que ya existe.
var ErrTemplateDuplicate = errors.New("notifications: template already exists for event/channel/locale")

// ErrPreferenceNotFound se devuelve cuando una preferencia no existe.
var ErrPreferenceNotFound = errors.New("notifications: preference not found")

// ErrPreferenceDuplicate se devuelve cuando se intenta crear una
// preferencia duplicada (user_id, event_type, channel).
var ErrPreferenceDuplicate = errors.New("notifications: preference already exists")

// ErrConsentNotFound se devuelve cuando un consentimiento no existe.
var ErrConsentNotFound = errors.New("notifications: consent not found")

// ErrConsentDuplicate se devuelve cuando se intenta crear un
// consentimiento duplicado (user_id, channel).
var ErrConsentDuplicate = errors.New("notifications: consent already exists for channel")

// ErrPushTokenNotFound se devuelve cuando un push token no existe.
var ErrPushTokenNotFound = errors.New("notifications: push token not found")

// ErrPushTokenDuplicate se devuelve cuando se intenta registrar un
// token duplicado (user_id, token).
var ErrPushTokenDuplicate = errors.New("notifications: push token already exists")

// ErrProviderConfigNotFound se devuelve cuando una config no existe.
var ErrProviderConfigNotFound = errors.New("notifications: provider config not found")

// ErrProviderConfigDuplicate se devuelve cuando se intenta crear una
// config duplicada (channel, provider_name).
var ErrProviderConfigDuplicate = errors.New("notifications: provider config already exists")

// ErrOutboxNotFound se devuelve cuando un mensaje del outbox no existe.
var ErrOutboxNotFound = errors.New("notifications: outbox entry not found")

// ErrOutboxDuplicate se devuelve cuando se viola la idempotencia del
// outbox (event_type, recipient_user_id, idempotency_key).
var ErrOutboxDuplicate = errors.New("notifications: outbox entry already exists (idempotency)")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio.
var ErrVersionConflict = errors.New("notifications: version conflict")

// ---------------------------------------------------------------------------
// TemplateRepository
// ---------------------------------------------------------------------------

// CreateTemplateInput agrupa los datos para persistir una plantilla nueva.
type CreateTemplateInput struct {
	EventType           string
	Channel             entities.Channel
	Locale              string
	Subject             *string
	BodyTemplate        string
	ProviderTemplateRef *string
	ActorID             string
}

// UpdateTemplateInput agrupa los datos para actualizar una plantilla.
type UpdateTemplateInput struct {
	ID                  string
	Subject             *string
	BodyTemplate        string
	ProviderTemplateRef *string
	ExpectedVersion     int32
	ActorID             string
}

// TemplateRepository es el puerto que persiste plantillas de notificacion.
type TemplateRepository interface {
	// Create inserta una plantilla nueva en estado 'active'.
	Create(ctx context.Context, in CreateTemplateInput) (entities.NotificationTemplate, error)
	// GetByID devuelve una plantilla por id. Si no existe, devuelve
	// ErrTemplateNotFound.
	GetByID(ctx context.Context, id string) (entities.NotificationTemplate, error)
	// List devuelve las plantillas activas (no soft-deleted) ordenadas
	// por event_type, channel.
	List(ctx context.Context) ([]entities.NotificationTemplate, error)
	// Update actualiza una plantilla existente con concurrencia optimista.
	Update(ctx context.Context, in UpdateTemplateInput) (entities.NotificationTemplate, error)
}

// ---------------------------------------------------------------------------
// PreferenceRepository
// ---------------------------------------------------------------------------

// UpsertPreferenceInput agrupa los datos para crear o actualizar una
// preferencia.
type UpsertPreferenceInput struct {
	UserID    string
	EventType string
	Channel   entities.Channel
	Enabled   bool
	ActorID   string
}

// PreferenceRepository es el puerto que persiste preferencias de
// notificacion por usuario.
type PreferenceRepository interface {
	// Upsert crea o actualiza una preferencia (user_id, event_type, channel).
	Upsert(ctx context.Context, in UpsertPreferenceInput) (entities.NotificationPreference, error)
	// ListByUserID devuelve las preferencias de un usuario.
	ListByUserID(ctx context.Context, userID string) ([]entities.NotificationPreference, error)
	// GetByUserEventChannel devuelve una preferencia especifica.
	GetByUserEventChannel(ctx context.Context, userID, eventType string, channel entities.Channel) (entities.NotificationPreference, error)
}

// ---------------------------------------------------------------------------
// ConsentRepository
// ---------------------------------------------------------------------------

// CreateConsentInput agrupa los datos para registrar un consentimiento.
type CreateConsentInput struct {
	UserID          string
	Channel         entities.Channel
	ConsentProofURL *string
	LegalBasis      *string
	ActorID         string
}

// ConsentRepository es el puerto que persiste consentimientos legales.
type ConsentRepository interface {
	// Create registra un consentimiento nuevo en estado 'active'.
	Create(ctx context.Context, in CreateConsentInput) (entities.NotificationConsent, error)
	// GetActiveByUserChannel devuelve el consentimiento activo de un
	// usuario para un canal. Si no existe, devuelve ErrConsentNotFound.
	GetActiveByUserChannel(ctx context.Context, userID string, channel entities.Channel) (entities.NotificationConsent, error)
	// Revoke revoca un consentimiento existente (soft revoke: establece
	// revoked_at y status='revoked'). Si no existe, devuelve
	// ErrConsentNotFound.
	Revoke(ctx context.Context, userID string, channel entities.Channel, actorID string) (entities.NotificationConsent, error)
	// ListByUserID devuelve todos los consentimientos de un usuario.
	ListByUserID(ctx context.Context, userID string) ([]entities.NotificationConsent, error)
}

// ---------------------------------------------------------------------------
// PushTokenRepository
// ---------------------------------------------------------------------------

// CreatePushTokenInput agrupa los datos para registrar un push token.
type CreatePushTokenInput struct {
	UserID   string
	Platform entities.Platform
	Token    string
	ActorID  string
}

// PushTokenRepository es el puerto que persiste push tokens.
type PushTokenRepository interface {
	// Create registra un push token nuevo.
	Create(ctx context.Context, in CreatePushTokenInput) (entities.NotificationPushToken, error)
	// GetByID devuelve un push token por id. Si no existe, devuelve
	// ErrPushTokenNotFound.
	GetByID(ctx context.Context, id string) (entities.NotificationPushToken, error)
	// ListActiveByUserID devuelve los tokens activos de un usuario.
	ListActiveByUserID(ctx context.Context, userID string) ([]entities.NotificationPushToken, error)
	// SoftDelete marca un push token como eliminado.
	SoftDelete(ctx context.Context, id string, actorID string) error
	// TouchLastSeen actualiza el last_seen_at de un token.
	TouchLastSeen(ctx context.Context, id string) error
}

// ---------------------------------------------------------------------------
// ProviderConfigRepository
// ---------------------------------------------------------------------------

// UpdateProviderConfigInput agrupa los datos para actualizar una config.
type UpdateProviderConfigInput struct {
	ID              string
	Config          []byte
	IsActive        bool
	ExpectedVersion int32
	ActorID         string
}

// ProviderConfigRepository es el puerto que persiste configuraciones de
// proveedores de envio.
type ProviderConfigRepository interface {
	// List devuelve todas las configs activas ordenadas por canal.
	List(ctx context.Context) ([]entities.NotificationProviderConfig, error)
	// GetActiveByChannel devuelve la config activa para un canal. Si no
	// existe, devuelve ErrProviderConfigNotFound.
	GetActiveByChannel(ctx context.Context, channel entities.Channel) (entities.NotificationProviderConfig, error)
	// Update actualiza una config con concurrencia optimista.
	Update(ctx context.Context, in UpdateProviderConfigInput) (entities.NotificationProviderConfig, error)
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un mensaje de
// notificacion.
type EnqueueOutboxInput struct {
	EventType       string
	RecipientUserID string
	Channel         entities.Channel
	Payload         []byte
	IdempotencyKey  string
	ScheduledAt     time.Time
	ActorID         string
}

// OutboxRepository es el puerto que persiste mensajes en el outbox de
// notificaciones.
type OutboxRepository interface {
	// Enqueue inserta un mensaje pendiente de envio.
	Enqueue(ctx context.Context, in EnqueueOutboxInput) (entities.NotificationOutbox, error)
	// GetByID devuelve un mensaje por id. Si no existe, devuelve
	// ErrOutboxNotFound.
	GetByID(ctx context.Context, id string) (entities.NotificationOutbox, error)
	// ListPending devuelve mensajes pendientes listos para procesar
	// (scheduled_at <= now() y status en pending/scheduled/failed_retry).
	ListPending(ctx context.Context, limit int32) ([]entities.NotificationOutbox, error)
	// MarkSending cambia status a 'sending' con concurrencia optimista.
	MarkSending(ctx context.Context, id string, expectedVersion int32) error
	// MarkSent marca un mensaje como enviado exitosamente.
	MarkSent(ctx context.Context, id string, expectedVersion int32) error
	// MarkFailedRetry registra un fallo con reintento: incrementa attempts,
	// fija last_error, reagenda scheduled_at.
	MarkFailedRetry(ctx context.Context, id string, expectedVersion int32, lastError string, nextAttemptAt time.Time) error
	// MarkFailedPermanent marca un mensaje como fallo permanente.
	MarkFailedPermanent(ctx context.Context, id string, expectedVersion int32, lastError string) error
	// MarkBlockedNoConsent marca como bloqueado por falta de consentimiento.
	MarkBlockedNoConsent(ctx context.Context, id string, expectedVersion int32) error
	// ListByRecipient devuelve los mensajes de un destinatario ordenados
	// por created_at desc.
	ListByRecipient(ctx context.Context, recipientUserID string) ([]entities.NotificationOutbox, error)
}

// ---------------------------------------------------------------------------
// DeliveryRepository
// ---------------------------------------------------------------------------

// CreateDeliveryInput agrupa los datos para registrar una entrega.
type CreateDeliveryInput struct {
	OutboxID          string
	ProviderName      string
	ProviderMessageID *string
	ProviderStatus    *string
	DeliveredAt       *time.Time
	FailureReason     *string
	ActorID           string
}

// DeliveryRepository es el puerto que persiste registros de entrega.
type DeliveryRepository interface {
	// Create inserta un registro de entrega.
	Create(ctx context.Context, in CreateDeliveryInput) (entities.NotificationDelivery, error)
	// ListByOutboxID devuelve los registros de entrega de un mensaje.
	ListByOutboxID(ctx context.Context, outboxID string) ([]entities.NotificationDelivery, error)
}
