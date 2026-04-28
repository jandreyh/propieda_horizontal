-- Queries del modulo people (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * La placa se persiste ya normalizada (uppercase + trim) por la capa
--     de aplicacion; las queries no transforman la entrada.
--   * Soft-delete via marcar status='archived' y deleted_at = now().
--   * Concurrencia optimista: cada update incrementa version.

-- name: CreateVehicle :one
-- Crea un vehiculo nuevo. La placa se asume ya normalizada (upper+trim).
INSERT INTO vehicles (
    plate, type, brand, model, color, year, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'active', $7, $7
)
RETURNING id, plate, type, brand, model, color, year, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetVehicleByID :one
-- Un vehiculo por id (solo activos / no eliminados).
SELECT id, plate, type, brand, model, color, year, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM vehicles
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetVehicleByPlate :one
-- Un vehiculo por placa (la placa se asume ya normalizada).
SELECT id, plate, type, brand, model, color, year, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM vehicles
 WHERE plate = $1
   AND deleted_at IS NULL;

-- name: ListAllVehicles :many
-- Lista todos los vehiculos activos ordenados por placa.
SELECT id, plate, type, brand, model, color, year, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM vehicles
 WHERE deleted_at IS NULL
 ORDER BY plate;

-- name: AssignVehicleToUnit :one
-- Crea una asignacion activa de un vehiculo a una unidad. La unicidad
-- (vehiculo asignado a UNA sola unidad activa) se enforza con el indice
-- parcial unique de la tabla; un INSERT que la viole devolvera error 23505
-- y la capa de repositorio lo mapea a ErrVehicleAlreadyAssigned.
INSERT INTO unit_vehicle_assignments (
    unit_id, vehicle_id, since_date, status,
    created_by, updated_by
) VALUES (
    sqlc.arg('unit_id'),
    sqlc.arg('vehicle_id'),
    COALESCE(sqlc.narg('since_date')::DATE, CURRENT_DATE),
    'active',
    sqlc.arg('created_by'),
    sqlc.arg('created_by')
)
RETURNING id, unit_id, vehicle_id, since_date, until_date, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListActiveAssignmentsByUnit :many
-- Lista asignaciones activas (sin until_date) de una unidad, junto con
-- los datos del vehiculo asociado para evitar un round-trip extra.
SELECT a.id, a.unit_id, a.vehicle_id, a.since_date, a.until_date, a.status,
       a.created_at, a.updated_at, a.deleted_at,
       a.created_by, a.updated_by, a.deleted_by, a.version,
       v.plate, v.type AS vehicle_type, v.brand, v.model, v.color, v.year
  FROM unit_vehicle_assignments a
  JOIN vehicles v ON v.id = a.vehicle_id
 WHERE a.unit_id = $1
   AND a.deleted_at IS NULL
   AND a.until_date IS NULL
 ORDER BY a.since_date DESC, a.created_at DESC;

-- name: EndVehicleAssignment :one
-- Cierra una asignacion fijando until_date = CURRENT_DATE (o el provisto).
-- Solo afecta filas activas (sin until_date).
UPDATE unit_vehicle_assignments
   SET until_date = COALESCE(sqlc.narg('until_date')::DATE, CURRENT_DATE),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND deleted_at IS NULL
   AND until_date IS NULL
RETURNING id, unit_id, vehicle_id, since_date, until_date, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;
