-- Queries del modulo tenant_config (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Branding es singleton: GetBranding devuelve la unica fila vigente.
--   * Soft-delete via marcar status='archived' y deleted_at = now().
--   * Concurrencia optimista: cada update incrementa version y compara
--     contra el version provisto.

-- name: ListSettings :many
-- Lista paginada de settings activos. Si @category es NULL/vacio, no
-- filtra por categoria.
SELECT id, key, value, description, category, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM tenant_settings
 WHERE deleted_at IS NULL
   AND (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category')::TEXT)
 ORDER BY category NULLS LAST, key
 LIMIT $1 OFFSET $2;

-- name: CountSettings :one
-- Total de settings activos para paginacion.
SELECT COUNT(*)
  FROM tenant_settings
 WHERE deleted_at IS NULL
   AND (sqlc.narg('category')::TEXT IS NULL OR category = sqlc.narg('category')::TEXT);

-- name: GetSetting :one
-- Una setting por key (solo activas).
SELECT id, key, value, description, category, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM tenant_settings
 WHERE key = $1
   AND deleted_at IS NULL;

-- name: UpsertSetting :one
-- Inserta o actualiza una setting por key. Si existia, incrementa version.
INSERT INTO tenant_settings (key, value, description, category, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, 'active', $5, $5)
ON CONFLICT (key) DO UPDATE
   SET value       = EXCLUDED.value,
       description = COALESCE(EXCLUDED.description, tenant_settings.description),
       category    = COALESCE(EXCLUDED.category,    tenant_settings.category),
       status      = 'active',
       deleted_at  = NULL,
       deleted_by  = NULL,
       updated_at  = now(),
       updated_by  = EXCLUDED.updated_by,
       version     = tenant_settings.version + 1
RETURNING id, key, value, description, category, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ArchiveSetting :one
-- Soft-delete: marca la setting como archived.
UPDATE tenant_settings
   SET status     = 'archived',
       deleted_at = now(),
       deleted_by = $2,
       updated_at = now(),
       updated_by = $2,
       version    = version + 1
 WHERE key = $1
   AND deleted_at IS NULL
RETURNING id, key, value, description, category, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetBranding :one
-- Devuelve la unica fila singleton de branding del tenant.
SELECT id, singleton, display_name, logo_url, primary_color, secondary_color,
       timezone, locale, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM tenant_branding
 WHERE deleted_at IS NULL
 LIMIT 1;

-- name: UpdateBranding :one
-- Actualiza la fila singleton. Concurrencia optimista por version.
UPDATE tenant_branding
   SET display_name    = $1,
       logo_url        = $2,
       primary_color   = $3,
       secondary_color = $4,
       timezone        = $5,
       locale          = $6,
       updated_at      = now(),
       updated_by      = $7,
       version         = version + 1
 WHERE singleton = TRUE
   AND deleted_at IS NULL
   AND version = $8
RETURNING id, singleton, display_name, logo_url, primary_color, secondary_color,
          timezone, locale, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;
