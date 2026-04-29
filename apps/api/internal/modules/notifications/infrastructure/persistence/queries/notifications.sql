-- Queries del modulo notifications (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.

-- ----------------------------------------------------------------------------
-- notification_templates
-- ----------------------------------------------------------------------------

-- name: CreateNotificationTemplate :one
-- Crea una plantilla nueva en estado 'active'.
INSERT INTO notification_templates (
    event_type, channel, locale, subject, body_template,
    provider_template_ref, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'active', $7, $7
)
RETURNING id, event_type, channel, locale, subject, body_template,
          provider_template_ref, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetNotificationTemplateByID :one
-- Devuelve una plantilla por id (no soft-deleted).
SELECT id, event_type, channel, locale, subject, body_template,
       provider_template_ref, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_templates
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListNotificationTemplates :many
-- Lista plantillas activas ordenadas por event_type, channel.
SELECT id, event_type, channel, locale, subject, body_template,
       provider_template_ref, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_templates
 WHERE deleted_at IS NULL
 ORDER BY event_type ASC, channel ASC;

-- name: UpdateNotificationTemplate :one
-- Actualiza una plantilla con concurrencia optimista.
UPDATE notification_templates
   SET subject              = sqlc.arg('new_subject'),
       body_template        = sqlc.arg('new_body_template'),
       provider_template_ref = sqlc.arg('new_provider_template_ref'),
       updated_at           = now(),
       updated_by           = sqlc.arg('updated_by'),
       version              = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, event_type, channel, locale, subject, body_template,
          provider_template_ref, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- notification_preferences
-- ----------------------------------------------------------------------------

-- name: UpsertNotificationPreference :one
-- Crea o actualiza una preferencia (ON CONFLICT upsert).
INSERT INTO notification_preferences (
    user_id, event_type, channel, enabled, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, 'active', $5, $5
)
ON CONFLICT (user_id, event_type, channel) WHERE deleted_at IS NULL
DO UPDATE SET enabled    = EXCLUDED.enabled,
              updated_at = now(),
              updated_by = EXCLUDED.updated_by,
              version    = notification_preferences.version + 1
RETURNING id, user_id, event_type, channel, enabled, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListNotificationPreferencesByUserID :many
-- Lista preferencias de un usuario (no soft-deleted).
SELECT id, user_id, event_type, channel, enabled, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_preferences
 WHERE user_id = $1
   AND deleted_at IS NULL
 ORDER BY event_type ASC, channel ASC;

-- name: GetNotificationPreferenceByUserEventChannel :one
-- Devuelve una preferencia especifica.
SELECT id, user_id, event_type, channel, enabled, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_preferences
 WHERE user_id = $1
   AND event_type = $2
   AND channel = $3
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- notification_consents
-- ----------------------------------------------------------------------------

-- name: CreateNotificationConsent :one
-- Registra un consentimiento nuevo en estado 'active'.
INSERT INTO notification_consents (
    user_id, channel, consented_at, consent_proof_url, legal_basis,
    status, created_by, updated_by
) VALUES (
    $1, $2, now(), $3, $4, 'active', $5, $5
)
RETURNING id, user_id, channel, consented_at, revoked_at,
          consent_proof_url, legal_basis, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetActiveNotificationConsentByUserChannel :one
-- Devuelve el consentimiento activo de un usuario para un canal.
SELECT id, user_id, channel, consented_at, revoked_at,
       consent_proof_url, legal_basis, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_consents
 WHERE user_id = $1
   AND channel = $2
   AND status = 'active'
   AND revoked_at IS NULL
   AND deleted_at IS NULL;

-- name: RevokeNotificationConsent :one
-- Revoca un consentimiento (establece revoked_at y status='revoked').
UPDATE notification_consents
   SET revoked_at  = now(),
       status      = 'revoked',
       updated_at  = now(),
       updated_by  = sqlc.arg('updated_by'),
       version     = version + 1
 WHERE user_id = sqlc.arg('user_id')
   AND channel = sqlc.arg('channel')
   AND status = 'active'
   AND revoked_at IS NULL
   AND deleted_at IS NULL
RETURNING id, user_id, channel, consented_at, revoked_at,
          consent_proof_url, legal_basis, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListNotificationConsentsByUserID :many
-- Lista consentimientos de un usuario (no soft-deleted).
SELECT id, user_id, channel, consented_at, revoked_at,
       consent_proof_url, legal_basis, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_consents
 WHERE user_id = $1
   AND deleted_at IS NULL
 ORDER BY channel ASC;

-- ----------------------------------------------------------------------------
-- notification_push_tokens
-- ----------------------------------------------------------------------------

-- name: CreateNotificationPushToken :one
-- Registra un push token nuevo.
INSERT INTO notification_push_tokens (
    user_id, platform, token, last_seen_at, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, now(), 'active', $4, $4
)
RETURNING id, user_id, platform, token, last_seen_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetNotificationPushTokenByID :one
-- Devuelve un push token por id (no soft-deleted).
SELECT id, user_id, platform, token, last_seen_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_push_tokens
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListActiveNotificationPushTokensByUserID :many
-- Lista tokens activos de un usuario.
SELECT id, user_id, platform, token, last_seen_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_push_tokens
 WHERE user_id = $1
   AND status = 'active'
   AND deleted_at IS NULL
 ORDER BY last_seen_at DESC;

-- name: SoftDeleteNotificationPushToken :exec
-- Soft delete de un push token.
UPDATE notification_push_tokens
   SET deleted_at  = now(),
       deleted_by  = sqlc.arg('deleted_by'),
       updated_at  = now(),
       updated_by  = sqlc.arg('deleted_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND deleted_at IS NULL;

-- name: TouchNotificationPushTokenLastSeen :exec
-- Actualiza el last_seen_at de un push token.
UPDATE notification_push_tokens
   SET last_seen_at = now(),
       updated_at   = now()
 WHERE id = $1
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- notification_provider_configs
-- ----------------------------------------------------------------------------

-- name: ListNotificationProviderConfigs :many
-- Lista provider configs (no soft-deleted) ordenadas por channel.
SELECT id, channel, provider_name, config, is_active, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_provider_configs
 WHERE deleted_at IS NULL
 ORDER BY channel ASC, provider_name ASC;

-- name: GetActiveNotificationProviderConfigByChannel :one
-- Devuelve la config activa para un canal.
SELECT id, channel, provider_name, config, is_active, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_provider_configs
 WHERE channel = $1
   AND is_active = true
   AND deleted_at IS NULL;

-- name: UpdateNotificationProviderConfig :one
-- Actualiza una config con concurrencia optimista.
UPDATE notification_provider_configs
   SET config     = sqlc.arg('new_config'),
       is_active  = sqlc.arg('new_is_active'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, channel, provider_name, config, is_active, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- notification_outbox
-- ----------------------------------------------------------------------------

-- name: EnqueueNotificationOutbox :one
-- Inserta un mensaje en el outbox de notificaciones.
INSERT INTO notification_outbox (
    event_type, recipient_user_id, channel, payload, idempotency_key,
    scheduled_at, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'pending', $7, $7
)
RETURNING id, event_type, recipient_user_id, channel, payload,
          idempotency_key, scheduled_at, sent_at, attempts,
          last_error, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetNotificationOutboxByID :one
-- Devuelve un mensaje del outbox por id (no soft-deleted).
SELECT id, event_type, recipient_user_id, channel, payload,
       idempotency_key, scheduled_at, sent_at, attempts,
       last_error, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_outbox
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListPendingNotificationOutbox :many
-- Lista mensajes pendientes listos para procesar.
SELECT id, event_type, recipient_user_id, channel, payload,
       idempotency_key, scheduled_at, sent_at, attempts,
       last_error, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_outbox
 WHERE deleted_at IS NULL
   AND status IN ('pending', 'scheduled', 'failed_retry')
   AND scheduled_at <= now()
 ORDER BY scheduled_at ASC
 LIMIT $1;

-- name: MarkNotificationOutboxSending :exec
-- Cambia status a 'sending' con concurrencia optimista.
UPDATE notification_outbox
   SET status     = 'sending',
       updated_at = now(),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- name: MarkNotificationOutboxSent :exec
-- Marca un mensaje como enviado exitosamente.
UPDATE notification_outbox
   SET status     = 'sent',
       sent_at    = now(),
       attempts   = attempts + 1,
       last_error = NULL,
       updated_at = now(),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- name: MarkNotificationOutboxFailedRetry :exec
-- Registra un fallo con reintento.
UPDATE notification_outbox
   SET status       = 'failed_retry',
       attempts     = attempts + 1,
       last_error   = sqlc.arg('last_error'),
       scheduled_at = sqlc.arg('next_attempt_at'),
       updated_at   = now(),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- name: MarkNotificationOutboxFailedPermanent :exec
-- Marca un mensaje como fallo permanente.
UPDATE notification_outbox
   SET status     = 'failed_permanent',
       attempts   = attempts + 1,
       last_error = sqlc.arg('last_error'),
       updated_at = now(),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- name: MarkNotificationOutboxBlockedNoConsent :exec
-- Marca como bloqueado por falta de consentimiento.
UPDATE notification_outbox
   SET status     = 'blocked_no_consent',
       updated_at = now(),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- name: ListNotificationOutboxByRecipient :many
-- Lista mensajes de un destinatario ordenados por created_at desc.
SELECT id, event_type, recipient_user_id, channel, payload,
       idempotency_key, scheduled_at, sent_at, attempts,
       last_error, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM notification_outbox
 WHERE recipient_user_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at DESC;

-- ----------------------------------------------------------------------------
-- notification_deliveries
-- ----------------------------------------------------------------------------

-- name: CreateNotificationDelivery :one
-- Inserta un registro de entrega.
INSERT INTO notification_deliveries (
    outbox_id, provider_name, provider_message_id,
    provider_status, delivered_at, failure_reason,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'submitted', $7, $7
)
RETURNING id, outbox_id, provider_name, provider_message_id,
          provider_status, delivered_at, failure_reason, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by;

-- name: ListNotificationDeliveriesByOutboxID :many
-- Lista registros de entrega de un mensaje.
SELECT id, outbox_id, provider_name, provider_message_id,
       provider_status, delivered_at, failure_reason, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM notification_deliveries
 WHERE outbox_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at DESC;
