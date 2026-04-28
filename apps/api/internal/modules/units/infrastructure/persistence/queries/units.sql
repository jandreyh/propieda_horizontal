-- Queries del modulo units.
--
-- Convenciones:
--   * Nombres en CamelCase con anotaciones :one|:many|:exec.
--   * SELECTs operativos siempre filtran deleted_at IS NULL.
--   * No hay columna tenant_id (la base entera es del tenant — CLAUDE.md).
--   * "Activo" para owners es until_date IS NULL; para occupants es
--     move_out_date IS NULL. Las terminaciones NO son soft-delete: se
--     setea la columna correspondiente para preservar historico.

-- name: CreateUnit :one
INSERT INTO units (
    structure_id, code, type, area_m2, bedrooms, coefficient,
    status, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    'active',
    $7
)
RETURNING id, structure_id, code, type, area_m2, bedrooms, coefficient,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version;

-- name: GetUnitByID :one
SELECT id, structure_id, code, type, area_m2, bedrooms, coefficient,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM units
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListUnitsByStructure :many
SELECT id, structure_id, code, type, area_m2, bedrooms, coefficient,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM units
WHERE structure_id = $1
  AND deleted_at IS NULL
ORDER BY code ASC;

-- name: ListAllActiveUnits :many
SELECT id, structure_id, code, type, area_m2, bedrooms, coefficient,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM units
WHERE deleted_at IS NULL
ORDER BY code ASC;

-- name: AddOwner :one
INSERT INTO unit_owners (
    unit_id, user_id, percentage, since_date, status, created_by
) VALUES (
    $1, $2, $3, $4, 'active', $5
)
RETURNING id, unit_id, user_id, percentage, since_date, until_date,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version;

-- name: ListOwnersByUnit :many
SELECT id, unit_id, user_id, percentage, since_date, until_date,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM unit_owners
WHERE unit_id = $1
  AND deleted_at IS NULL
  AND until_date IS NULL
ORDER BY since_date ASC;

-- name: TerminateOwnership :exec
UPDATE unit_owners
SET until_date = $2::date,
    updated_at = now(),
    updated_by = $3,
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL
  AND until_date IS NULL;

-- name: AddOccupant :one
INSERT INTO unit_occupancies (
    unit_id, user_id, role_in_unit, is_primary, move_in_date,
    status, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    'active', $6
)
RETURNING id, unit_id, user_id, role_in_unit, is_primary,
    move_in_date, move_out_date,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version;

-- name: ListActiveOccupantsByUnit :many
SELECT id, unit_id, user_id, role_in_unit, is_primary,
    move_in_date, move_out_date,
    status, created_at, updated_at, deleted_at,
    created_by, updated_by, deleted_by, version
FROM unit_occupancies
WHERE unit_id = $1
  AND deleted_at IS NULL
  AND move_out_date IS NULL
ORDER BY move_in_date ASC;

-- name: MoveOutOccupant :exec
UPDATE unit_occupancies
SET move_out_date = $2::date,
    updated_at = now(),
    updated_by = $3,
    version = version + 1
WHERE id = $1
  AND deleted_at IS NULL
  AND move_out_date IS NULL;

-- name: GetActivePeopleForUnit :many
-- Devuelve owners activos + occupants activos en una sola query.
-- "Owners activos" = unit_owners sin until_date y no soft-deleted.
-- "Occupants activos" = unit_occupancies sin move_out_date y no soft-deleted.
-- El JOIN con users obtiene el nombre completo y el documento.
-- El campo role_in_unit unifica owners (etiqueta 'owner') y occupants
-- (su rol nativo). is_primary solo aplica a occupants; en owners viene
-- siempre false. since_date corresponde a owner.since_date u
-- occupant.move_in_date segun la fuente.
SELECT
    u.id AS user_id,
    (u.names || ' ' || u.last_names) AS full_name,
    (u.document_type || ':' || u.document_number) AS document,
    'owner'::text AS role_in_unit,
    false::boolean AS is_primary,
    o.since_date::date AS since_date
FROM unit_owners o
JOIN users u ON u.id = o.user_id AND u.deleted_at IS NULL
WHERE o.unit_id = $1
  AND o.deleted_at IS NULL
  AND o.until_date IS NULL
UNION ALL
SELECT
    u.id AS user_id,
    (u.names || ' ' || u.last_names) AS full_name,
    (u.document_type || ':' || u.document_number) AS document,
    oc.role_in_unit::text AS role_in_unit,
    oc.is_primary AS is_primary,
    oc.move_in_date::date AS since_date
FROM unit_occupancies oc
JOIN users u ON u.id = oc.user_id AND u.deleted_at IS NULL
WHERE oc.unit_id = $1
  AND oc.deleted_at IS NULL
  AND oc.move_out_date IS NULL
ORDER BY since_date ASC;
