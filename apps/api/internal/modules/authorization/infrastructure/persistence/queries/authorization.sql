-- queries/authorization.sql
--
-- Conjunto de queries del modulo authorization. sqlc genera el paquete
-- `authzdb` en `internal/modules/authorization/infrastructure/persistence/sqlcgen`.
--
-- Convenciones:
--   - Soft delete via deleted_at IS NULL en lecturas.
--   - Concurrencia optimista en updates de roles (WHERE version = $).
--   - user_role_assignments NO tiene deleted_at: se materializan
--     revocaciones via revoked_at + revocation_reason.

-- name: ListActiveRoles :many
SELECT id, name, description, is_system, status, version,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
FROM roles
WHERE deleted_at IS NULL
  AND status = 'active'
ORDER BY name;

-- name: GetRoleByID :one
SELECT id, name, description, is_system, status, version,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
FROM roles
WHERE id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetRoleByName :one
SELECT id, name, description, is_system, status, version,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
FROM roles
WHERE name = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: CreateRole :one
INSERT INTO roles (name, description, is_system, status, created_by, updated_by)
VALUES ($1, $2, false, 'active', $3, $3)
RETURNING id, name, description, is_system, status, version,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by;

-- name: UpdateRoleName :one
UPDATE roles
SET name        = $2,
    description = $3,
    updated_by  = $4,
    updated_at  = now(),
    version     = version + 1
WHERE id = $1
  AND deleted_at IS NULL
  AND is_system = false
  AND version = $5
RETURNING id, name, description, is_system, status, version,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by;

-- name: ArchiveRole :execrows
UPDATE roles
SET status      = 'archived',
    deleted_at  = now(),
    deleted_by  = $2,
    updated_by  = $2,
    updated_at  = now(),
    version     = version + 1
WHERE id = $1
  AND deleted_at IS NULL
  AND is_system = false;

-- name: ListPermissions :many
SELECT id, namespace, description, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
FROM permissions
WHERE deleted_at IS NULL
  AND status = 'active'
ORDER BY namespace;

-- name: GetPermissionByNamespace :one
SELECT id, namespace, description, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
FROM permissions
WHERE namespace = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetPermissionByID :one
SELECT id, namespace, description, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
FROM permissions
WHERE id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListPermissionsForRole :many
SELECT p.id, p.namespace, p.description, p.status,
       p.created_at, p.updated_at, p.deleted_at,
       p.created_by, p.updated_by, p.deleted_by
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE rp.role_id = $1
  AND p.deleted_at IS NULL
ORDER BY p.namespace;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: RevokePermissionFromRole :execrows
DELETE FROM role_permissions
WHERE role_id = $1
  AND permission_id = $2;

-- name: ClearPermissionsForRole :execrows
DELETE FROM role_permissions
WHERE role_id = $1;

-- name: CreateAssignment :one
INSERT INTO user_role_assignments (
    user_id, role_id, scope_type, scope_id,
    granted_by, status, created_by, updated_by
)
VALUES ($1, $2, $3, $4, $5, 'active', $5, $5)
RETURNING id, user_id, role_id, scope_type, scope_id,
          granted_by, granted_at, revoked_at, revocation_reason,
          status, version, created_at, updated_at,
          created_by, updated_by;

-- name: RevokeAssignment :execrows
UPDATE user_role_assignments
SET revoked_at        = now(),
    revocation_reason = $2,
    status            = 'revoked',
    updated_by        = $3,
    updated_at        = now(),
    version           = version + 1
WHERE id = $1
  AND revoked_at IS NULL;

-- name: GetActiveAssignmentsByUser :many
SELECT id, user_id, role_id, scope_type, scope_id,
       granted_by, granted_at, revoked_at, revocation_reason,
       status, version, created_at, updated_at,
       created_by, updated_by
FROM user_role_assignments
WHERE user_id = $1
  AND revoked_at IS NULL
ORDER BY granted_at DESC;

-- name: ListPermissionsForUser :many
SELECT DISTINCT p.namespace
FROM user_role_assignments ura
JOIN roles r          ON r.id = ura.role_id
                      AND r.deleted_at IS NULL
                      AND r.status = 'active'
JOIN role_permissions rp ON rp.role_id = r.id
JOIN permissions p    ON p.id = rp.permission_id
                      AND p.deleted_at IS NULL
                      AND p.status = 'active'
WHERE ura.user_id = $1
  AND ura.revoked_at IS NULL
  AND ura.status = 'active'
ORDER BY p.namespace;
