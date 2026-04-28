-- Queries del modulo residential_structure (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Soft-delete via marcar status='archived' y deleted_at = now().
--   * Concurrencia optimista: cada update incrementa version y compara
--     contra el version provisto (WHERE version = $expected).

-- name: ListActiveStructures :many
-- Lista todas las estructuras activas ordenadas por order_index, name.
SELECT id, name, type, parent_id, description, order_index, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM residential_structures
 WHERE deleted_at IS NULL
 ORDER BY order_index, name;

-- name: GetStructureByID :one
-- Una estructura por id (solo activas).
SELECT id, name, type, parent_id, description, order_index, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM residential_structures
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: CreateStructure :one
-- Crea una estructura nueva.
INSERT INTO residential_structures (
    name, type, parent_id, description, order_index, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, 'active', $6, $6
)
RETURNING id, name, type, parent_id, description, order_index, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: UpdateStructure :one
-- Actualiza una estructura usando concurrencia optimista. Si la version
-- no coincide o la fila no existe (o esta archivada), la query no
-- devuelve filas y el repo distingue ambos casos.
UPDATE residential_structures
   SET name        = $1,
       type        = $2,
       parent_id   = $3,
       description = $4,
       order_index = $5,
       updated_at  = now(),
       updated_by  = $6,
       version     = version + 1
 WHERE id = $7
   AND deleted_at IS NULL
   AND version = $8
RETURNING id, name, type, parent_id, description, order_index, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ArchiveStructure :one
-- Soft-delete: marca la estructura como archived.
UPDATE residential_structures
   SET status     = 'archived',
       deleted_at = now(),
       deleted_by = $2,
       updated_at = now(),
       updated_by = $2,
       version    = version + 1
 WHERE id = $1
   AND deleted_at IS NULL
RETURNING id, name, type, parent_id, description, order_index, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListChildStructures :many
-- Lista los hijos directos de una estructura padre.
SELECT id, name, type, parent_id, description, order_index, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM residential_structures
 WHERE parent_id = $1
   AND deleted_at IS NULL
 ORDER BY order_index, name;
