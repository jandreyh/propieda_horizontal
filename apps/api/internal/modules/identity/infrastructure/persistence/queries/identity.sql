-- Queries del modulo identity.
--
-- Convenciones:
--   * Nombres en CamelCase con anotaciones :one|:many|:exec.
--   * SELECTs siempre filtran deleted_at IS NULL para users (soft delete).
--   * No hay columna tenant_id (la base entera es del tenant — CLAUDE.md).
--   * Hashing de refresh token: sha256 hex (calculado en Go).

-- name: GetUserByID :one
SELECT
    id, document_type, document_number, names, last_names,
    email, phone, password_hash, mfa_secret, mfa_enrolled_at,
    failed_login_attempts, locked_until, last_login_at,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT
    id, document_type, document_number, names, last_names,
    email, phone, password_hash, mfa_secret, mfa_enrolled_at,
    failed_login_attempts, locked_until, last_login_at,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: GetUserByDocument :one
SELECT
    id, document_type, document_number, names, last_names,
    email, phone, password_hash, mfa_secret, mfa_enrolled_at,
    failed_login_attempts, locked_until, last_login_at,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM users
WHERE document_type = $1
  AND document_number = $2
  AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (
    document_type, document_number, names, last_names,
    email, phone, password_hash, mfa_secret, mfa_enrolled_at,
    status, created_by
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9,
    COALESCE(NULLIF($10, ''), 'active'),
    $11
)
RETURNING id, document_type, document_number, names, last_names,
    email, phone, password_hash, mfa_secret, mfa_enrolled_at,
    failed_login_attempts, locked_until, last_login_at,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version;

-- name: IncrementFailedAttempts :one
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
    updated_at = now(),
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL
RETURNING failed_login_attempts;

-- name: ResetFailedAttempts :exec
UPDATE users
SET failed_login_attempts = 0,
    updated_at = now(),
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL;

-- name: LockUser :exec
UPDATE users
SET locked_until = $2,
    failed_login_attempts = 0,
    updated_at = now(),
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL;

-- name: UnlockUser :exec
UPDATE users
SET locked_until = NULL,
    failed_login_attempts = 0,
    updated_at = now(),
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL;

-- name: UpdateLastLoginAt :exec
UPDATE users
SET last_login_at = $2,
    updated_at = now(),
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL;

-- name: CreateSession :one
INSERT INTO user_sessions (
    user_id, token_hash, parent_session_id,
    ip, user_agent, issued_at, expires_at, status
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7,
    'active'
)
RETURNING id, user_id, token_hash, parent_session_id,
    ip, user_agent, issued_at, expires_at,
    revoked_at, revocation_reason, status,
    created_at, updated_at, created_by, updated_by, version;

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, parent_session_id,
    ip, user_agent, issued_at, expires_at,
    revoked_at, revocation_reason, status,
    created_at, updated_at, created_by, updated_by, version
FROM user_sessions
WHERE token_hash = $1;

-- name: GetSessionByID :one
SELECT id, user_id, token_hash, parent_session_id,
    ip, user_agent, issued_at, expires_at,
    revoked_at, revocation_reason, status,
    created_at, updated_at, created_by, updated_by, version
FROM user_sessions
WHERE id = $1;

-- name: RevokeSession :exec
UPDATE user_sessions
SET revoked_at = $2,
    revocation_reason = $3,
    status = 'revoked',
    updated_at = now(),
    version = version + 1
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeSessionChain :exec
WITH RECURSIVE chain AS (
    SELECT us0.id AS sess_id, us0.parent_session_id AS sess_parent
    FROM user_sessions us0
    WHERE us0.id = $1
    UNION
    SELECT s.id, s.parent_session_id
    FROM user_sessions s
    JOIN chain c ON s.parent_session_id = c.sess_id
    UNION
    SELECT s.id, s.parent_session_id
    FROM user_sessions s
    JOIN chain c ON s.id = c.sess_parent
)
UPDATE user_sessions us
SET revoked_at = $2,
    revocation_reason = $3,
    status = 'revoked',
    updated_at = now(),
    version = version + 1
FROM chain
WHERE us.id = chain.sess_id
  AND us.revoked_at IS NULL;

-- name: ListUnusedRecoveryCodes :many
SELECT id, user_id, code_hash, used_at, status,
    created_at, updated_at, created_by, updated_by, version
FROM user_mfa_recovery_codes
WHERE user_id = $1
  AND used_at IS NULL
ORDER BY created_at ASC;

-- name: MarkRecoveryCodeUsed :exec
UPDATE user_mfa_recovery_codes
SET used_at = $2,
    status = 'used',
    updated_at = now(),
    version = version + 1
WHERE user_id = $1
  AND code_hash = $3
  AND used_at IS NULL;
