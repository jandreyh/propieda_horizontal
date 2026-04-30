-- Queries del modulo platform_identity (DB central). Sigue ADR 0007.

-- name: GetPlatformUserByEmail :one
SELECT *
FROM platform_users
WHERE lower(email) = lower($1)
  AND deleted_at IS NULL;

-- name: GetPlatformUserByDocument :one
SELECT *
FROM platform_users
WHERE document_type = $1
  AND document_number = $2
  AND deleted_at IS NULL;

-- name: GetPlatformUserByID :one
SELECT *
FROM platform_users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetPlatformUserByPublicCode :one
SELECT *
FROM platform_users
WHERE public_code = $1
  AND deleted_at IS NULL;

-- name: CreatePlatformUser :one
INSERT INTO platform_users (
    document_type, document_number, names, last_names,
    email, phone, password_hash, public_code, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, 'active'))
RETURNING *;

-- name: UpdatePlatformUserLastLogin :exec
UPDATE platform_users
SET last_login_at = $2,
    failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = now()
WHERE id = $1;

-- name: IncrementFailedLogin :one
UPDATE platform_users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts + 1 >= 5 THEN now() + INTERVAL '15 minutes'
        ELSE locked_until
    END,
    updated_at = now()
WHERE id = $1
RETURNING failed_login_attempts, locked_until;

-- name: SuspendPlatformUser :exec
UPDATE platform_users
SET status = 'suspended',
    updated_at = now()
WHERE id = $1;

-- name: ListMembershipsForUser :many
-- Indice central de membresias. La tabla platform_user_memberships
-- mantiene una proyeccion liviana del role del usuario en cada tenant.
-- La verdad ultima esta en tenant_user_links de cada tenant DB.
SELECT
    t.id          AS tenant_id,
    t.slug        AS tenant_slug,
    t.display_name AS tenant_name,
    t.logo_url,
    t.primary_color,
    m.role,
    m.status      AS membership_status
FROM platform_user_memberships m
JOIN tenants t ON t.id = m.tenant_id
WHERE m.platform_user_id = $1
  AND m.status = 'active'
  AND t.status = 'active'
ORDER BY t.display_name
LIMIT 50;

-- name: UpsertMembership :one
INSERT INTO platform_user_memberships (
    platform_user_id, tenant_id, role, status
)
VALUES ($1, $2, $3, COALESCE($4, 'active'))
ON CONFLICT (platform_user_id, tenant_id) DO UPDATE
    SET role = EXCLUDED.role,
        status = EXCLUDED.status,
        updated_at = now()
RETURNING *;

-- name: BlockMembership :exec
UPDATE platform_user_memberships
SET status = 'blocked',
    updated_at = now()
WHERE platform_user_id = $1
  AND tenant_id = $2;

-- name: GetMembership :one
SELECT *
FROM platform_user_memberships
WHERE platform_user_id = $1
  AND tenant_id = $2;

-- name: HasMembership :one
SELECT EXISTS(
    SELECT 1 FROM platform_user_memberships m
    JOIN tenants t ON t.id = m.tenant_id
    WHERE m.platform_user_id = $1
      AND t.slug = $2
      AND m.status = 'active'
      AND t.status = 'active'
) AS has_access;

-- name: GetTenantBySlug :one
SELECT *
FROM tenants
WHERE slug = $1
  AND status = 'active';

-- name: GetTenantByID :one
SELECT *
FROM tenants
WHERE id = $1;

-- name: ListTenants :many
SELECT *
FROM tenants
WHERE status != 'archived'
ORDER BY display_name;

-- name: CreatePlatformUserSession :one
INSERT INTO platform_user_sessions (
    platform_user_id, token_hash, ip, user_agent, expires_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT *
FROM platform_user_sessions
WHERE token_hash = $1
  AND status = 'active'
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RevokeSession :exec
UPDATE platform_user_sessions
SET status = 'revoked',
    revoked_at = now(),
    revocation_reason = $2,
    updated_at = now()
WHERE id = $1;

-- name: RevokeAllUserSessions :exec
UPDATE platform_user_sessions
SET status = 'revoked',
    revoked_at = now(),
    revocation_reason = $2,
    updated_at = now()
WHERE platform_user_id = $1
  AND revoked_at IS NULL;

-- name: RegisterPushDevice :one
INSERT INTO platform_push_devices (
    platform_user_id, device_token, platform, device_label
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (platform_user_id, device_token) DO UPDATE
    SET last_seen_at = now(),
        revoked_at = NULL,
        platform = EXCLUDED.platform,
        device_label = EXCLUDED.device_label
RETURNING *;

-- name: RevokePushDevice :exec
UPDATE platform_push_devices
SET revoked_at = now()
WHERE id = $1
  AND platform_user_id = $2;

-- name: ListPushDevicesForUser :many
SELECT *
FROM platform_push_devices
WHERE platform_user_id = $1
  AND revoked_at IS NULL
ORDER BY last_seen_at DESC;
