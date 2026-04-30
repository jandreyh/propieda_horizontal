-- Identidad global de plataforma (ADR 0007).
--
-- Una persona = una fila aqui, sin importar en cuantos conjuntos opere.
-- Las tablas `users` por tenant pasan a ser proyecciones (`tenant_user_links`)
-- que apuntan a este id por su columna `platform_user_id`.

CREATE TABLE IF NOT EXISTS platform_users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type          TEXT         NOT NULL,
    document_number        TEXT         NOT NULL,
    names                  TEXT         NOT NULL,
    last_names             TEXT         NOT NULL,
    email                  TEXT         NOT NULL,
    phone                  TEXT         NULL,
    photo_url              TEXT         NULL,
    password_hash          TEXT         NOT NULL,
    mfa_secret             TEXT         NULL,
    mfa_enrolled_at        TIMESTAMPTZ  NULL,
    public_code            TEXT         NOT NULL,
    failed_login_attempts  INTEGER      NOT NULL DEFAULT 0,
    locked_until           TIMESTAMPTZ  NULL,
    last_login_at          TIMESTAMPTZ  NULL,
    status                 TEXT         NOT NULL DEFAULT 'active',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ  NULL,
    CONSTRAINT platform_users_email_unique UNIQUE (email),
    CONSTRAINT platform_users_document_unique UNIQUE (document_type, document_number),
    CONSTRAINT platform_users_public_code_unique UNIQUE (public_code),
    CONSTRAINT platform_users_status_chk CHECK (status IN ('active', 'suspended')),
    CONSTRAINT platform_users_doctype_chk CHECK (document_type IN ('CC','CE','PA','TI','RC','NIT'))
);

CREATE INDEX IF NOT EXISTS platform_users_email_idx
    ON platform_users (lower(email))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS platform_users_public_code_idx
    ON platform_users (public_code)
    WHERE deleted_at IS NULL;

-- Sesiones globales de plataforma. Reemplazan a user_sessions del tenant DB.
-- token_hash es SHA-256 del refresh_token plano para soportar revocation.
CREATE TABLE IF NOT EXISTS platform_user_sessions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    token_hash          TEXT         NOT NULL UNIQUE,
    parent_session_id   UUID         NULL REFERENCES platform_user_sessions(id) ON DELETE SET NULL,
    ip                  INET         NULL,
    user_agent          TEXT         NULL,
    issued_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ  NOT NULL,
    revoked_at          TIMESTAMPTZ  NULL,
    revocation_reason   TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT platform_user_sessions_status_chk
        CHECK (status IN ('active', 'revoked', 'expired'))
);

CREATE INDEX IF NOT EXISTS platform_user_sessions_user_idx
    ON platform_user_sessions (platform_user_id, status)
    WHERE revoked_at IS NULL;

-- Codigos de recuperacion MFA (uno por persona, no por tenant).
CREATE TABLE IF NOT EXISTS platform_user_mfa_recovery_codes (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id  UUID         NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    code_hash         TEXT         NOT NULL,
    used_at           TIMESTAMPTZ  NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT platform_mfa_recovery_unique UNIQUE (platform_user_id, code_hash)
);

CREATE INDEX IF NOT EXISTS platform_user_mfa_recovery_user_idx
    ON platform_user_mfa_recovery_codes (platform_user_id)
    WHERE used_at IS NULL;
