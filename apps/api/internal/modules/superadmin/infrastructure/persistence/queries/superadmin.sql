-- Queries del modulo superadmin (DB central).

-- name: CreateTenant :one
INSERT INTO tenants (
    slug, display_name, database_url, plan, status,
    administrator_id, logo_url, primary_color,
    timezone, country, currency, expected_units, activated_at
)
VALUES (
    $1, $2, $3, COALESCE($4, 'pilot'), COALESCE($5, 'provisioning'),
    $6, $7, $8,
    COALESCE($9, 'America/Bogota'), COALESCE($10, 'CO'),
    COALESCE($11, 'COP'), $12, $13
)
RETURNING *;

-- name: UpdateTenantStatus :exec
UPDATE tenants
SET status = $2,
    activated_at = CASE
        WHEN $2 = 'active' AND activated_at IS NULL THEN now()
        ELSE activated_at
    END,
    suspended_at = CASE
        WHEN $2 = 'suspended' THEN now()
        ELSE suspended_at
    END,
    updated_at = now()
WHERE id = $1;

-- name: DeleteTenant :exec
DELETE FROM tenants
WHERE id = $1;

-- name: SearchPlatformUsersByEmail :many
SELECT *
FROM platform_users
WHERE lower(email) LIKE lower('%' || $1 || '%')
  AND deleted_at IS NULL
ORDER BY email
LIMIT 50;

-- name: SearchPlatformUsersByDocument :many
SELECT *
FROM platform_users
WHERE document_type = $1
  AND document_number LIKE $2 || '%'
  AND deleted_at IS NULL
ORDER BY last_names, names
LIMIT 50;

-- name: CreateAdministrator :one
INSERT INTO platform_administrators (
    name, legal_id, contact_email, contact_phone, status
)
VALUES ($1, $2, $3, $4, COALESCE($5, 'active'))
RETURNING *;

-- name: ListAdministrators :many
SELECT *
FROM platform_administrators
WHERE status != 'inactive'
ORDER BY name;

-- name: GetAdministratorByID :one
SELECT *
FROM platform_administrators
WHERE id = $1;

-- name: AssignTenantToAdministrator :exec
UPDATE tenants
SET administrator_id = $2,
    updated_at = now()
WHERE id = $1;

-- name: ListTenantsByAdministrator :many
SELECT *
FROM tenants
WHERE administrator_id = $1
  AND status != 'archived'
ORDER BY display_name;

-- name: InsertAuditLog :exec
INSERT INTO platform_audit_logs (
    actor_user_id, action, target_type, target_id, metadata, ip, user_agent
)
VALUES ($1, $2, $3, $4, $5, $6, $7);
