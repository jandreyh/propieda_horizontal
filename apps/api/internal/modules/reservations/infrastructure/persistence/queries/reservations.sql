-- Queries del modulo reservations (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- common_areas
-- ----------------------------------------------------------------------------

-- name: CreateCommonArea :one
-- Crea una zona comun nueva en estado 'active'.
INSERT INTO common_areas (
    code, name, kind, max_capacity,
    opening_time, closing_time, slot_duration_minutes,
    cost_per_use, security_deposit,
    requires_approval, is_active, description,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
    'active', $13, $13
)
RETURNING id, code, name, kind, max_capacity,
          opening_time, closing_time, slot_duration_minutes,
          cost_per_use, security_deposit,
          requires_approval, is_active, description,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetCommonAreaByID :one
-- Devuelve una zona comun por id (no soft-deleted).
SELECT id, code, name, kind, max_capacity,
       opening_time, closing_time, slot_duration_minutes,
       cost_per_use, security_deposit,
       requires_approval, is_active, description,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM common_areas
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListCommonAreas :many
-- Lista zonas comunes activas (no soft-deleted) ordenadas por name.
SELECT id, code, name, kind, max_capacity,
       opening_time, closing_time, slot_duration_minutes,
       cost_per_use, security_deposit,
       requires_approval, is_active, description,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM common_areas
 WHERE deleted_at IS NULL
 ORDER BY name ASC;

-- name: UpdateCommonArea :one
-- Actualiza una zona comun con concurrencia optimista.
UPDATE common_areas
   SET code                 = sqlc.arg('new_code'),
       name                 = sqlc.arg('new_name'),
       kind                 = sqlc.arg('new_kind'),
       max_capacity         = sqlc.arg('new_max_capacity'),
       opening_time         = sqlc.arg('new_opening_time'),
       closing_time         = sqlc.arg('new_closing_time'),
       slot_duration_minutes = sqlc.arg('new_slot_duration_minutes'),
       cost_per_use         = sqlc.arg('new_cost_per_use'),
       security_deposit     = sqlc.arg('new_security_deposit'),
       requires_approval    = sqlc.arg('new_requires_approval'),
       is_active            = sqlc.arg('new_is_active'),
       description          = sqlc.arg('new_description'),
       status               = sqlc.arg('new_status'),
       updated_at           = now(),
       updated_by           = sqlc.arg('updated_by'),
       version              = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, code, name, kind, max_capacity,
          opening_time, closing_time, slot_duration_minutes,
          cost_per_use, security_deposit,
          requires_approval, is_active, description,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- reservation_blackouts
-- ----------------------------------------------------------------------------

-- name: CreateBlackout :one
-- Crea un bloqueo temporal en una zona comun.
INSERT INTO reservation_blackouts (
    common_area_id, from_at, to_at, reason,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, 'active', $5, $5
)
RETURNING id, common_area_id, from_at, to_at, reason,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListActiveBlackoutsByCommonArea :many
-- Lista bloqueos activos de una zona comun.
SELECT id, common_area_id, from_at, to_at, reason,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservation_blackouts
 WHERE common_area_id = $1
   AND status = 'active'
   AND deleted_at IS NULL
 ORDER BY from_at ASC;

-- name: ListBlackoutsByCommonAreaAndWindow :many
-- Lista bloqueos activos de una zona comun que solapan con una ventana.
SELECT id, common_area_id, from_at, to_at, reason,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservation_blackouts
 WHERE common_area_id = $1
   AND status = 'active'
   AND deleted_at IS NULL
   AND from_at < $3
   AND to_at > $2
 ORDER BY from_at ASC;

-- ----------------------------------------------------------------------------
-- reservations
-- ----------------------------------------------------------------------------

-- name: CreateReservation :one
-- Crea una reserva de zona comun.
INSERT INTO reservations (
    common_area_id, unit_id, requested_by_user_id,
    slot_start_at, slot_end_at, attendees_count,
    cost, security_deposit, qr_code_hash,
    idempotency_key, notes,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $3, $3
)
RETURNING id, common_area_id, unit_id, requested_by_user_id,
          slot_start_at, slot_end_at, attendees_count,
          cost, security_deposit, deposit_refunded,
          qr_code_hash, idempotency_key, notes,
          approved_by, approved_at, cancelled_by, cancelled_at,
          consumed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetReservationByID :one
-- Devuelve una reserva por id (no soft-deleted).
SELECT id, common_area_id, unit_id, requested_by_user_id,
       slot_start_at, slot_end_at, attendees_count,
       cost, security_deposit, deposit_refunded,
       qr_code_hash, idempotency_key, notes,
       approved_by, approved_at, cancelled_by, cancelled_at,
       consumed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservations
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetReservationByIdempotencyKey :one
-- Devuelve una reserva por clave de idempotencia.
SELECT id, common_area_id, unit_id, requested_by_user_id,
       slot_start_at, slot_end_at, attendees_count,
       cost, security_deposit, deposit_refunded,
       qr_code_hash, idempotency_key, notes,
       approved_by, approved_at, cancelled_by, cancelled_at,
       consumed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservations
 WHERE idempotency_key = $1
   AND deleted_at IS NULL;

-- name: GetReservationByQRCodeHash :one
-- Devuelve una reserva por hash de QR.
SELECT id, common_area_id, unit_id, requested_by_user_id,
       slot_start_at, slot_end_at, attendees_count,
       cost, security_deposit, deposit_refunded,
       qr_code_hash, idempotency_key, notes,
       approved_by, approved_at, cancelled_by, cancelled_at,
       consumed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservations
 WHERE qr_code_hash = $1
   AND deleted_at IS NULL;

-- name: ListReservations :many
-- Lista todas las reservas (no soft-deleted) ordenadas por slot_start_at desc.
SELECT id, common_area_id, unit_id, requested_by_user_id,
       slot_start_at, slot_end_at, attendees_count,
       cost, security_deposit, deposit_refunded,
       qr_code_hash, idempotency_key, notes,
       approved_by, approved_at, cancelled_by, cancelled_at,
       consumed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservations
 WHERE deleted_at IS NULL
 ORDER BY slot_start_at DESC;

-- name: ListReservationsByUnit :many
-- Lista reservas de una unidad ordenadas por slot_start_at desc.
SELECT id, common_area_id, unit_id, requested_by_user_id,
       slot_start_at, slot_end_at, attendees_count,
       cost, security_deposit, deposit_refunded,
       qr_code_hash, idempotency_key, notes,
       approved_by, approved_at, cancelled_by, cancelled_at,
       consumed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservations
 WHERE unit_id = $1
   AND deleted_at IS NULL
 ORDER BY slot_start_at DESC;

-- name: ListReservationsByCommonAreaAndDate :many
-- Lista reservas de una zona comun en un rango de fecha.
SELECT id, common_area_id, unit_id, requested_by_user_id,
       slot_start_at, slot_end_at, attendees_count,
       cost, security_deposit, deposit_refunded,
       qr_code_hash, idempotency_key, notes,
       approved_by, approved_at, cancelled_by, cancelled_at,
       consumed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM reservations
 WHERE common_area_id = $1
   AND slot_start_at >= $2
   AND slot_start_at < $3
   AND deleted_at IS NULL
 ORDER BY slot_start_at ASC;

-- name: UpdateReservationStatus :one
-- Actualiza el status de una reserva con concurrencia optimista.
UPDATE reservations
   SET status     = sqlc.arg('new_status'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, common_area_id, unit_id, requested_by_user_id,
          slot_start_at, slot_end_at, attendees_count,
          cost, security_deposit, deposit_refunded,
          qr_code_hash, idempotency_key, notes,
          approved_by, approved_at, cancelled_by, cancelled_at,
          consumed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ApproveReservation :one
-- Aprueba una reserva (status='confirmed', registra approved_by/at).
UPDATE reservations
   SET status      = 'confirmed',
       approved_by = sqlc.arg('approved_by'),
       approved_at = now(),
       updated_at  = now(),
       updated_by  = sqlc.arg('approved_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status = 'pending'
RETURNING id, common_area_id, unit_id, requested_by_user_id,
          slot_start_at, slot_end_at, attendees_count,
          cost, security_deposit, deposit_refunded,
          qr_code_hash, idempotency_key, notes,
          approved_by, approved_at, cancelled_by, cancelled_at,
          consumed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: CancelReservation :one
-- Cancela una reserva (status='cancelled', registra cancelled_by/at).
UPDATE reservations
   SET status       = 'cancelled',
       cancelled_by = sqlc.arg('cancelled_by'),
       cancelled_at = now(),
       updated_at   = now(),
       updated_by   = sqlc.arg('cancelled_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status IN ('pending', 'confirmed')
RETURNING id, common_area_id, unit_id, requested_by_user_id,
          slot_start_at, slot_end_at, attendees_count,
          cost, security_deposit, deposit_refunded,
          qr_code_hash, idempotency_key, notes,
          approved_by, approved_at, cancelled_by, cancelled_at,
          consumed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: RejectReservation :one
-- Rechaza una reserva pendiente (status='rejected').
UPDATE reservations
   SET status     = 'rejected',
       updated_at = now(),
       updated_by = sqlc.arg('rejected_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status = 'pending'
RETURNING id, common_area_id, unit_id, requested_by_user_id,
          slot_start_at, slot_end_at, attendees_count,
          cost, security_deposit, deposit_refunded,
          qr_code_hash, idempotency_key, notes,
          approved_by, approved_at, cancelled_by, cancelled_at,
          consumed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: CheckinReservation :one
-- Registra checkin de una reserva confirmada (status='consumed').
UPDATE reservations
   SET status      = 'consumed',
       consumed_at = now(),
       updated_at  = now(),
       updated_by  = sqlc.arg('guard_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status = 'confirmed'
RETURNING id, common_area_id, unit_id, requested_by_user_id,
          slot_start_at, slot_end_at, attendees_count,
          cost, security_deposit, deposit_refunded,
          qr_code_hash, idempotency_key, notes,
          approved_by, approved_at, cancelled_by, cancelled_at,
          consumed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- reservation_status_history (append-only)
-- ----------------------------------------------------------------------------

-- name: RecordStatusHistory :one
-- Inserta un registro append-only de cambio de estado.
INSERT INTO reservation_status_history (
    reservation_id, from_status, to_status,
    changed_by, reason
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, reservation_id, from_status, to_status,
          changed_by, reason, changed_at;

-- name: ListStatusHistoryByReservation :many
-- Historial de cambios de estado de una reserva.
SELECT id, reservation_id, from_status, to_status,
       changed_by, reason, changed_at
  FROM reservation_status_history
 WHERE reservation_id = $1
 ORDER BY changed_at DESC;

-- ----------------------------------------------------------------------------
-- reservations_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueueReservationOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO reservations_outbox_events (
    aggregate_id, event_type, payload, next_attempt_at, attempts
) VALUES (
    $1, $2, $3, now(), 0
)
RETURNING id, aggregate_id, event_type, payload, created_at,
          next_attempt_at, attempts, delivered_at, last_error;

-- name: LockPendingReservationOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, aggregate_id, event_type, payload, created_at,
       next_attempt_at, attempts, delivered_at, last_error
  FROM reservations_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkReservationOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE reservations_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkReservationOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE reservations_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
