-- Tenant DB: modulo notifications (Fase 15 — multicanal: email, push,
-- WhatsApp, SMS).
--
-- Crea las tablas operativas:
--   * notification_templates           : plantillas por (event,channel,locale).
--   * notification_preferences         : preferencias por usuario y evento.
--   * notification_consents            : opt-in legal por canal.
--   * notification_push_tokens         : tokens push FCM/APNs.
--   * notification_provider_configs    : configuracion de proveedores.
--   * notification_outbox              : cola de envios (at-least-once).
--   * notification_deliveries          : registro de entrega por proveedor.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO tenant_id.
--   * Campos estandar + soft delete.
--   * notification_outbox: UNIQUE
--     (event_type, recipient_user_id, idempotency_key) WHERE
--     deleted_at IS NULL para idempotencia.

-- ----------------------------------------------------------------------------
-- notification_templates
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_templates (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type                  TEXT         NOT NULL,
    channel                     TEXT         NOT NULL,
    locale                      TEXT         NOT NULL DEFAULT 'es-CO',
    subject                     TEXT         NULL,
    body_template               TEXT         NOT NULL,
    provider_template_ref       TEXT         NULL,
    status                      TEXT         NOT NULL DEFAULT 'active',
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ  NULL,
    created_by                  UUID         NULL REFERENCES users(id),
    updated_by                  UUID         NULL REFERENCES users(id),
    deleted_by                  UUID         NULL REFERENCES users(id),
    version                     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT notification_templates_channel_chk
        CHECK (channel IN ('email', 'push', 'whatsapp', 'sms')),
    CONSTRAINT notification_templates_status_chk
        CHECK (status IN ('active', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS notification_templates_unique
    ON notification_templates (event_type, channel, locale)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- notification_preferences
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_preferences (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type                  TEXT         NOT NULL,
    channel                     TEXT         NOT NULL,
    enabled                     BOOLEAN      NOT NULL DEFAULT true,
    status                      TEXT         NOT NULL DEFAULT 'active',
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ  NULL,
    created_by                  UUID         NULL REFERENCES users(id),
    updated_by                  UUID         NULL REFERENCES users(id),
    deleted_by                  UUID         NULL REFERENCES users(id),
    version                     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT notification_preferences_channel_chk
        CHECK (channel IN ('email', 'push', 'whatsapp', 'sms')),
    CONSTRAINT notification_preferences_status_chk
        CHECK (status IN ('active', 'archived'))
);

-- Una preferencia (enabled/disabled) por (user, event, channel).
CREATE UNIQUE INDEX IF NOT EXISTS notification_preferences_unique
    ON notification_preferences (user_id, event_type, channel)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS notification_preferences_user_idx
    ON notification_preferences (user_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- notification_consents (opt-in legal)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_consents (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel             TEXT         NOT NULL,
    consented_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ  NULL,
    consent_proof_url   TEXT         NULL,
    legal_basis         TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT notification_consents_channel_chk
        CHECK (channel IN ('email', 'push', 'whatsapp', 'sms')),
    CONSTRAINT notification_consents_status_chk
        CHECK (status IN ('active', 'revoked'))
);

-- Un consentimiento activo por (user, channel).
CREATE UNIQUE INDEX IF NOT EXISTS notification_consents_unique
    ON notification_consents (user_id, channel)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- notification_push_tokens
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_push_tokens (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform        TEXT         NOT NULL,
    token           TEXT         NOT NULL,
    last_seen_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT notification_push_tokens_platform_chk
        CHECK (platform IN ('ios', 'android', 'web')),
    CONSTRAINT notification_push_tokens_status_chk
        CHECK (status IN ('active', 'invalid', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS notification_push_tokens_unique
    ON notification_push_tokens (user_id, token)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS notification_push_tokens_user_active_idx
    ON notification_push_tokens (user_id)
    WHERE deleted_at IS NULL AND status = 'active';

-- ----------------------------------------------------------------------------
-- notification_provider_configs
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_provider_configs (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    channel         TEXT         NOT NULL,
    provider_name   TEXT         NOT NULL,
    config          JSONB        NOT NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT notification_provider_configs_channel_chk
        CHECK (channel IN ('email', 'push', 'whatsapp', 'sms')),
    CONSTRAINT notification_provider_configs_status_chk
        CHECK (status IN ('active', 'archived'))
);

-- Solo una configuracion (channel, provider_name) viva.
CREATE UNIQUE INDEX IF NOT EXISTS notification_provider_configs_unique
    ON notification_provider_configs (channel, provider_name)
    WHERE deleted_at IS NULL;

-- Solo un provider activo por canal.
CREATE UNIQUE INDEX IF NOT EXISTS notification_provider_configs_one_active_idx
    ON notification_provider_configs (channel)
    WHERE deleted_at IS NULL AND is_active = true;

-- ----------------------------------------------------------------------------
-- notification_outbox
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_outbox (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type               TEXT         NOT NULL,
    recipient_user_id        UUID         NOT NULL REFERENCES users(id),
    channel                  TEXT         NOT NULL,
    payload                  JSONB        NOT NULL,
    idempotency_key          TEXT         NOT NULL,
    scheduled_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    sent_at                  TIMESTAMPTZ  NULL,
    attempts                 INTEGER      NOT NULL DEFAULT 0,
    last_error               TEXT         NULL,
    status                   TEXT         NOT NULL DEFAULT 'pending',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT notification_outbox_channel_chk
        CHECK (channel IN ('email', 'push', 'whatsapp', 'sms')),
    CONSTRAINT notification_outbox_status_chk
        CHECK (status IN ('pending', 'scheduled', 'sending', 'sent',
                          'failed_retry', 'failed_permanent',
                          'blocked_no_consent', 'cancelled')),
    CONSTRAINT notification_outbox_attempts_chk
        CHECK (attempts >= 0)
);

-- Idempotencia: un evento por (event_type, recipient, idempotency_key).
CREATE UNIQUE INDEX IF NOT EXISTS notification_outbox_idempotency_unique
    ON notification_outbox (event_type, recipient_user_id, idempotency_key)
    WHERE deleted_at IS NULL;

-- Worker hot path: pendientes/programados/reintentos listos para procesar.
CREATE INDEX IF NOT EXISTS notification_outbox_worker_idx
    ON notification_outbox (scheduled_at)
    WHERE deleted_at IS NULL
          AND status IN ('pending', 'scheduled', 'failed_retry');

-- Reportes por canal y dia.
CREATE INDEX IF NOT EXISTS notification_outbox_channel_status_idx
    ON notification_outbox (channel, status, created_at)
    WHERE deleted_at IS NULL;

-- Lookup por destinatario.
CREATE INDEX IF NOT EXISTS notification_outbox_recipient_idx
    ON notification_outbox (recipient_user_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- notification_deliveries
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_deliveries (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    outbox_id                UUID         NOT NULL REFERENCES notification_outbox(id) ON DELETE CASCADE,
    provider_name            TEXT         NOT NULL,
    provider_message_id      TEXT         NULL,
    provider_status          TEXT         NULL,
    delivered_at             TIMESTAMPTZ  NULL,
    failure_reason           TEXT         NULL,
    status                   TEXT         NOT NULL DEFAULT 'submitted',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    CONSTRAINT notification_deliveries_status_chk
        CHECK (status IN ('submitted', 'delivered', 'failed', 'unknown'))
);

CREATE INDEX IF NOT EXISTS notification_deliveries_outbox_idx
    ON notification_deliveries (outbox_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS notification_deliveries_provider_idx
    ON notification_deliveries (provider_name, status, created_at)
    WHERE deleted_at IS NULL;
