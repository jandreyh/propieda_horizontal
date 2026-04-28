-- Queries del modulo parking (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- parking_spaces
-- ----------------------------------------------------------------------------

-- name: CreateParkingSpace :one
-- Crea un espacio nuevo en estado 'active'.
INSERT INTO parking_spaces (
    code, type, structure_id, level, zone, monthly_fee,
    is_visitor, notes, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 'active', $9, $9
)
RETURNING id, code, type, structure_id, level, zone, monthly_fee,
          is_visitor, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetParkingSpaceByID :one
-- Devuelve un espacio por id (no soft-deleted).
SELECT id, code, type, structure_id, level, zone, monthly_fee,
       is_visitor, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_spaces
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetParkingSpaceByCode :one
-- Devuelve un espacio por codigo (activo, no soft-deleted).
SELECT id, code, type, structure_id, level, zone, monthly_fee,
       is_visitor, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_spaces
 WHERE code = $1
   AND deleted_at IS NULL;

-- name: ListParkingSpaces :many
-- Lista espacios activos (no soft-deleted) ordenados por codigo.
SELECT id, code, type, structure_id, level, zone, monthly_fee,
       is_visitor, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_spaces
 WHERE deleted_at IS NULL
 ORDER BY code ASC;

-- name: UpdateParkingSpace :one
-- Actualiza un espacio con concurrencia optimista.
UPDATE parking_spaces
   SET code         = sqlc.arg('new_code'),
       type         = sqlc.arg('new_type'),
       structure_id = sqlc.arg('new_structure_id'),
       level        = sqlc.arg('new_level'),
       zone         = sqlc.arg('new_zone'),
       monthly_fee  = sqlc.arg('new_monthly_fee'),
       is_visitor   = sqlc.arg('new_is_visitor'),
       notes        = sqlc.arg('new_notes'),
       status       = sqlc.arg('new_status'),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, code, type, structure_id, level, zone, monthly_fee,
          is_visitor, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SoftDeleteParkingSpace :exec
-- Soft delete de un espacio con concurrencia optimista.
UPDATE parking_spaces
   SET deleted_at  = now(),
       deleted_by  = sqlc.arg('deleted_by'),
       updated_at  = now(),
       updated_by  = sqlc.arg('deleted_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- parking_assignments
-- ----------------------------------------------------------------------------

-- name: CreateParkingAssignment :one
-- Crea una asignacion nueva en estado 'active'.
INSERT INTO parking_assignments (
    parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
    since_date, notes, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'active', $4, $4
)
RETURNING id, parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
          since_date, until_date, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetParkingAssignmentByID :one
-- Devuelve una asignacion por id (no soft-deleted).
SELECT id, parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
       since_date, until_date, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_assignments
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetActiveAssignmentBySpaceID :one
-- Devuelve la asignacion activa para un espacio (until_date IS NULL).
SELECT id, parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
       since_date, until_date, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_assignments
 WHERE parking_space_id = $1
   AND until_date IS NULL
   AND deleted_at IS NULL
 LIMIT 1;

-- name: ListActiveAssignmentsByUnitID :many
-- Lista asignaciones activas de una unidad.
SELECT id, parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
       since_date, until_date, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_assignments
 WHERE unit_id = $1
   AND until_date IS NULL
   AND deleted_at IS NULL
 ORDER BY since_date DESC;

-- name: ListAssignmentsBySpaceID :many
-- Lista todas las asignaciones de un espacio (activas y cerradas).
SELECT id, parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
       since_date, until_date, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_assignments
 WHERE parking_space_id = $1
   AND deleted_at IS NULL
 ORDER BY since_date DESC;

-- name: CloseAssignment :one
-- Cierra una asignacion (establece until_date y status='closed').
UPDATE parking_assignments
   SET until_date  = sqlc.arg('until_date'),
       status      = 'closed',
       updated_at  = now(),
       updated_by  = sqlc.arg('updated_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, parking_space_id, unit_id, vehicle_id, assigned_by_user_id,
          since_date, until_date, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SoftDeleteAssignment :exec
-- Soft delete de una asignacion con concurrencia optimista.
UPDATE parking_assignments
   SET deleted_at  = now(),
       deleted_by  = sqlc.arg('deleted_by'),
       updated_at  = now(),
       updated_by  = sqlc.arg('deleted_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- parking_assignment_history
-- ----------------------------------------------------------------------------

-- name: RecordAssignmentHistory :one
-- Inserta un registro append-only de historial de asignacion.
INSERT INTO parking_assignment_history (
    parking_space_id, unit_id, assignment_id,
    since_date, until_date, closed_reason,
    snapshot_payload, recorded_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, parking_space_id, unit_id, assignment_id,
          since_date, until_date, closed_reason,
          snapshot_payload, recorded_at, recorded_by;

-- name: ListAssignmentHistoryBySpaceID :many
-- Historial de un espacio ordenado por recorded_at desc.
SELECT id, parking_space_id, unit_id, assignment_id,
       since_date, until_date, closed_reason,
       snapshot_payload, recorded_at, recorded_by
  FROM parking_assignment_history
 WHERE parking_space_id = $1
 ORDER BY recorded_at DESC;

-- name: ListAssignmentHistoryByUnitID :many
-- Historial de una unidad ordenado por recorded_at desc.
SELECT id, parking_space_id, unit_id, assignment_id,
       since_date, until_date, closed_reason,
       snapshot_payload, recorded_at, recorded_by
  FROM parking_assignment_history
 WHERE unit_id = $1
 ORDER BY recorded_at DESC;

-- ----------------------------------------------------------------------------
-- parking_visitor_reservations
-- ----------------------------------------------------------------------------

-- name: CreateVisitorReservation :one
-- Crea una reserva de visitante.
INSERT INTO parking_visitor_reservations (
    parking_space_id, unit_id, requested_by,
    visitor_name, visitor_document, vehicle_plate,
    slot_start_at, slot_end_at, idempotency_key,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    'confirmed', $3, $3
)
RETURNING id, parking_space_id, unit_id, requested_by,
          visitor_name, visitor_document, vehicle_plate,
          slot_start_at, slot_end_at, idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetVisitorReservationByID :one
-- Devuelve una reserva por id (no soft-deleted).
SELECT id, parking_space_id, unit_id, requested_by,
       visitor_name, visitor_document, vehicle_plate,
       slot_start_at, slot_end_at, idempotency_key, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_visitor_reservations
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetVisitorReservationByIdempotencyKey :one
-- Devuelve una reserva por clave de idempotencia.
SELECT id, parking_space_id, unit_id, requested_by,
       visitor_name, visitor_document, vehicle_plate,
       slot_start_at, slot_end_at, idempotency_key, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_visitor_reservations
 WHERE idempotency_key = $1
   AND deleted_at IS NULL;

-- name: ListVisitorReservationsByDate :many
-- Lista reservas en un rango de fecha.
SELECT id, parking_space_id, unit_id, requested_by,
       visitor_name, visitor_document, vehicle_plate,
       slot_start_at, slot_end_at, idempotency_key, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_visitor_reservations
 WHERE slot_start_at >= $1
   AND slot_start_at < $2
   AND deleted_at IS NULL
 ORDER BY slot_start_at ASC;

-- name: ListVisitorReservationsByUnit :many
-- Lista reservas de una unidad.
SELECT id, parking_space_id, unit_id, requested_by,
       visitor_name, visitor_document, vehicle_plate,
       slot_start_at, slot_end_at, idempotency_key, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_visitor_reservations
 WHERE unit_id = $1
   AND deleted_at IS NULL
 ORDER BY slot_start_at DESC;

-- name: UpdateVisitorReservationStatus :one
-- Actualiza el status de una reserva con concurrencia optimista.
UPDATE parking_visitor_reservations
   SET status     = sqlc.arg('new_status'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, parking_space_id, unit_id, requested_by,
          visitor_name, visitor_document, vehicle_plate,
          slot_start_at, slot_end_at, idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: CancelVisitorReservation :one
-- Cancela una reserva (atajo con status='cancelled').
UPDATE parking_visitor_reservations
   SET status     = 'cancelled',
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status IN ('pending', 'confirmed')
RETURNING id, parking_space_id, unit_id, requested_by,
          visitor_name, visitor_document, vehicle_plate,
          slot_start_at, slot_end_at, idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- parking_lottery_runs
-- ----------------------------------------------------------------------------

-- name: CreateLotteryRun :one
-- Crea un sorteo en estado 'completed'.
INSERT INTO parking_lottery_runs (
    name, seed_hash, criteria, executed_by,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, 'completed', $4, $4
)
RETURNING id, name, seed_hash, criteria, executed_at, executed_by,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetLotteryRunByID :one
-- Devuelve un sorteo por id (no soft-deleted).
SELECT id, name, seed_hash, criteria, executed_at, executed_by,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_lottery_runs
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListLotteryRuns :many
-- Lista sorteos ordenados por executed_at desc.
SELECT id, name, seed_hash, criteria, executed_at, executed_by,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_lottery_runs
 WHERE deleted_at IS NULL
 ORDER BY executed_at DESC;

-- ----------------------------------------------------------------------------
-- parking_lottery_results
-- ----------------------------------------------------------------------------

-- name: CreateLotteryResult :one
-- Crea un resultado individual de sorteo.
INSERT INTO parking_lottery_results (
    lottery_run_id, unit_id, parking_space_id, position,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $6
)
RETURNING id, lottery_run_id, unit_id, parking_space_id, position,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListLotteryResultsByRunID :many
-- Lista resultados de un sorteo ordenados por posicion.
SELECT id, lottery_run_id, unit_id, parking_space_id, position,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM parking_lottery_results
 WHERE lottery_run_id = $1
   AND deleted_at IS NULL
 ORDER BY position ASC;

-- ----------------------------------------------------------------------------
-- parking_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueueParkingOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO parking_outbox_events (
    aggregate_id, event_type, payload, next_attempt_at, attempts
) VALUES (
    $1, $2, $3, now(), 0
)
RETURNING id, aggregate_id, event_type, payload, created_at,
          next_attempt_at, attempts, delivered_at, last_error;

-- name: LockPendingParkingOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, aggregate_id, event_type, payload, created_at,
       next_attempt_at, attempts, delivered_at, last_error
  FROM parking_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkParkingOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE parking_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkParkingOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE parking_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
