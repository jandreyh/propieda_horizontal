-- Tenant DB: modulo identity.
--
-- Crea las tres tablas operativas del modulo de identidad:
--   * users                    : actores autenticables del tenant.
--   * user_sessions            : sesiones activas / refresh tokens.
--   * user_mfa_recovery_codes  : codigos de recuperacion MFA single-use.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * Identidad de negocio compuesta: (document_type, document_number).
--   * Email es opcional pero UNIQUE-when-not-null.
--   * Hash de contrasena en columna password_hash (argon2id encoded).

CREATE TABLE IF NOT EXISTS users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type          TEXT         NOT NULL,
    document_number        TEXT         NOT NULL,
    names                  TEXT         NOT NULL,
    last_names             TEXT         NOT NULL,
    email                  TEXT         NULL,
    phone                  TEXT         NULL,
    password_hash          TEXT         NOT NULL,
    mfa_secret             TEXT         NULL,
    mfa_enrolled_at        TIMESTAMPTZ  NULL,
    failed_login_attempts  INTEGER      NOT NULL DEFAULT 0,
    locked_until           TIMESTAMPTZ  NULL,
    last_login_at          TIMESTAMPTZ  NULL,
    status                 TEXT         NOT NULL DEFAULT 'active',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ  NULL,
    created_by             UUID         NULL REFERENCES users(id),
    updated_by             UUID         NULL REFERENCES users(id),
    deleted_by             UUID         NULL REFERENCES users(id),
    version                INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT users_status_chk
        CHECK (status IN ('active', 'inactive', 'suspended')),
    CONSTRAINT users_document_type_chk
        CHECK (document_type IN ('CC', 'CE', 'PA', 'TI', 'RC', 'NIT')),
    CONSTRAINT users_document_unique
        UNIQUE (document_type, document_number)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique
    ON users (email)
    WHERE email IS NOT NULL;

CREATE INDEX IF NOT EXISTS users_status_idx
    ON users (status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS user_sessions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash          TEXT         NOT NULL UNIQUE,
    parent_session_id   UUID         NULL REFERENCES user_sessions(id) ON DELETE SET NULL,
    ip                  INET         NULL,
    user_agent          TEXT         NULL,
    issued_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ  NOT NULL,
    revoked_at          TIMESTAMPTZ  NULL,
    revocation_reason   TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT user_sessions_status_chk
        CHECK (status IN ('active', 'revoked', 'expired'))
);

CREATE INDEX IF NOT EXISTS user_sessions_active_idx
    ON user_sessions (user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS user_sessions_parent_idx
    ON user_sessions (parent_session_id)
    WHERE parent_session_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_mfa_recovery_codes (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT         NOT NULL,
    used_at     TIMESTAMPTZ  NULL,
    status      TEXT         NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by  UUID         NULL REFERENCES users(id),
    updated_by  UUID         NULL REFERENCES users(id),
    version     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT user_mfa_recovery_codes_status_chk
        CHECK (status IN ('active', 'used', 'revoked'))
);

CREATE INDEX IF NOT EXISTS user_mfa_recovery_codes_user_idx
    ON user_mfa_recovery_codes (user_id)
    WHERE used_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS user_mfa_recovery_codes_hash_unique
    ON user_mfa_recovery_codes (user_id, code_hash);
