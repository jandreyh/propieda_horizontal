-- Push devices a nivel plataforma (ADR 0007 seccion 10).
-- Las notificaciones se enrutan a la persona, no al tenant. Cada conjunto
-- emite eventos con `recipient_platform_user_id` y un worker central
-- dispara FCM/APNs al device correcto.

CREATE TABLE IF NOT EXISTS platform_push_devices (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    device_token        TEXT         NOT NULL,
    platform            TEXT         NOT NULL,
    device_label        TEXT         NULL,
    last_seen_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ  NULL,
    CONSTRAINT platform_push_devices_platform_chk
        CHECK (platform IN ('ios','android','web')),
    CONSTRAINT platform_push_devices_token_unique
        UNIQUE (platform_user_id, device_token)
);

CREATE INDEX IF NOT EXISTS platform_push_devices_user_idx
    ON platform_push_devices (platform_user_id)
    WHERE revoked_at IS NULL;
