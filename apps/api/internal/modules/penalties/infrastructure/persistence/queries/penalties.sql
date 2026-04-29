-- Queries del modulo penalties (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- penalty_catalog
-- ----------------------------------------------------------------------------

-- name: CreatePenaltyCatalogEntry :one
-- Crea una entrada del catalogo en estado 'active'.
INSERT INTO penalty_catalog (
    code, name, description, default_sanction_type,
    base_amount, recurrence_multiplier, recurrence_cap_multiplier,
    requires_council_threshold, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 'active', $9, $9
)
RETURNING id, code, name, description, default_sanction_type,
          base_amount, recurrence_multiplier, recurrence_cap_multiplier,
          requires_council_threshold, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPenaltyCatalogByID :one
-- Devuelve una entrada del catalogo por id (no soft-deleted).
SELECT id, code, name, description, default_sanction_type,
       base_amount, recurrence_multiplier, recurrence_cap_multiplier,
       requires_council_threshold, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM penalty_catalog
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListPenaltyCatalog :many
-- Lista entradas activas del catalogo ordenadas por code.
SELECT id, code, name, description, default_sanction_type,
       base_amount, recurrence_multiplier, recurrence_cap_multiplier,
       requires_council_threshold, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM penalty_catalog
 WHERE deleted_at IS NULL
 ORDER BY code ASC;

-- name: UpdatePenaltyCatalogEntry :one
-- Actualiza una entrada del catalogo con concurrencia optimista.
UPDATE penalty_catalog
   SET code                       = sqlc.arg('new_code'),
       name                       = sqlc.arg('new_name'),
       description                = sqlc.arg('new_description'),
       default_sanction_type      = sqlc.arg('new_default_sanction_type'),
       base_amount                = sqlc.arg('new_base_amount'),
       recurrence_multiplier      = sqlc.arg('new_recurrence_multiplier'),
       recurrence_cap_multiplier  = sqlc.arg('new_recurrence_cap_multiplier'),
       requires_council_threshold = sqlc.arg('new_requires_council_threshold'),
       status                     = sqlc.arg('new_status'),
       updated_at                 = now(),
       updated_by                 = sqlc.arg('updated_by'),
       version                    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, code, name, description, default_sanction_type,
          base_amount, recurrence_multiplier, recurrence_cap_multiplier,
          requires_council_threshold, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- penalties
-- ----------------------------------------------------------------------------

-- name: CreatePenalty :one
-- Crea una sancion en estado 'drafted'.
INSERT INTO penalties (
    catalog_id, debtor_user_id, unit_id, source_incident_id,
    sanction_type, amount, reason, imposed_by_user_id,
    requires_council_approval, idempotency_key,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    'drafted', $8, $8
)
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPenaltyByID :one
-- Devuelve una sancion por id (no soft-deleted).
SELECT id, catalog_id, debtor_user_id, unit_id, source_incident_id,
       sanction_type, amount, reason, imposed_by_user_id,
       notified_at, appeal_deadline_at, confirmed_at,
       settled_at, dismissed_at, cancelled_at,
       requires_council_approval,
       council_approved_by_user_id, council_approved_at,
       idempotency_key, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM penalties
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListPenalties :many
-- Lista sanciones (no soft-deleted) ordenadas por created_at desc.
SELECT id, catalog_id, debtor_user_id, unit_id, source_incident_id,
       sanction_type, amount, reason, imposed_by_user_id,
       notified_at, appeal_deadline_at, confirmed_at,
       settled_at, dismissed_at, cancelled_at,
       requires_council_approval,
       council_approved_by_user_id, council_approved_at,
       idempotency_key, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM penalties
 WHERE deleted_at IS NULL
 ORDER BY created_at DESC;

-- name: UpdatePenaltyStatus :one
-- Actualiza el status de una sancion con concurrencia optimista.
UPDATE penalties
   SET status     = sqlc.arg('new_status'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPenaltyNotified :one
-- Marca una sancion como notificada.
UPDATE penalties
   SET status             = 'notified',
       notified_at        = sqlc.arg('notified_at'),
       appeal_deadline_at = sqlc.arg('appeal_deadline_at'),
       updated_at         = now(),
       updated_by         = sqlc.arg('updated_by'),
       version            = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPenaltyCouncilApproved :one
-- Registra la aprobacion del consejo.
UPDATE penalties
   SET council_approved_by_user_id = sqlc.arg('council_approved_by'),
       council_approved_at         = sqlc.arg('council_approved_at'),
       updated_at                  = now(),
       updated_by                  = sqlc.arg('updated_by'),
       version                     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPenaltyConfirmed :one
-- Confirma una sancion.
UPDATE penalties
   SET status       = 'confirmed',
       confirmed_at = sqlc.arg('confirmed_at'),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPenaltySettled :one
-- Salda una sancion.
UPDATE penalties
   SET status     = 'settled',
       settled_at = sqlc.arg('settled_at'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPenaltyDismissed :one
-- Desestima una sancion (apelacion aceptada).
UPDATE penalties
   SET status       = 'dismissed',
       dismissed_at = sqlc.arg('dismissed_at'),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPenaltyCancelled :one
-- Cancela una sancion.
UPDATE penalties
   SET status       = 'cancelled',
       cancelled_at = sqlc.arg('cancelled_at'),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, catalog_id, debtor_user_id, unit_id, source_incident_id,
          sanction_type, amount, reason, imposed_by_user_id,
          notified_at, appeal_deadline_at, confirmed_at,
          settled_at, dismissed_at, cancelled_at,
          requires_council_approval,
          council_approved_by_user_id, council_approved_at,
          idempotency_key, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: CountPenaltyReincidence :one
-- Cuenta sanciones confirmed/settled para el mismo (debtor, catalog)
-- en una ventana temporal.
SELECT count(*)::int AS cnt
  FROM penalties
 WHERE debtor_user_id = $1
   AND catalog_id = $2
   AND confirmed_at >= $3
   AND status IN ('confirmed', 'settled')
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- penalty_appeals
-- ----------------------------------------------------------------------------

-- name: CreatePenaltyAppeal :one
-- Crea una apelacion en estado 'submitted'.
INSERT INTO penalty_appeals (
    penalty_id, submitted_by_user_id, grounds,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, 'submitted', $2, $2
)
RETURNING id, penalty_id, submitted_by_user_id, submitted_at,
          grounds, resolved_by_user_id, resolved_at, resolution,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPenaltyAppealByID :one
-- Devuelve una apelacion por id (no soft-deleted).
SELECT id, penalty_id, submitted_by_user_id, submitted_at,
       grounds, resolved_by_user_id, resolved_at, resolution,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM penalty_appeals
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetActiveAppealByPenaltyID :one
-- Devuelve la apelacion activa de un penalty (submitted o under_review).
SELECT id, penalty_id, submitted_by_user_id, submitted_at,
       grounds, resolved_by_user_id, resolved_at, resolution,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM penalty_appeals
 WHERE penalty_id = $1
   AND status IN ('submitted', 'under_review')
   AND deleted_at IS NULL
 LIMIT 1;

-- name: ResolvePenaltyAppeal :one
-- Resuelve una apelacion con concurrencia optimista.
UPDATE penalty_appeals
   SET resolved_by_user_id = sqlc.arg('resolved_by'),
       resolved_at         = now(),
       resolution          = sqlc.arg('resolution'),
       status              = sqlc.arg('new_status'),
       updated_at          = now(),
       updated_by          = sqlc.arg('updated_by'),
       version             = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, penalty_id, submitted_by_user_id, submitted_at,
          grounds, resolved_by_user_id, resolved_at, resolution,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- penalty_status_history
-- ----------------------------------------------------------------------------

-- name: RecordPenaltyStatusHistory :one
-- Inserta un registro append-only de historial de transiciones.
INSERT INTO penalty_status_history (
    penalty_id, from_status, to_status,
    transitioned_by_user_id, notes,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5,
    'recorded', $4, $4
)
RETURNING id, penalty_id, from_status, to_status,
          transitioned_by_user_id, transitioned_at, notes,
          status, created_at, updated_at,
          created_by, updated_by;

-- name: ListPenaltyStatusHistory :many
-- Historial de un penalty ordenado por transitioned_at desc.
SELECT id, penalty_id, from_status, to_status,
       transitioned_by_user_id, transitioned_at, notes,
       status, created_at, updated_at,
       created_by, updated_by
  FROM penalty_status_history
 WHERE penalty_id = $1
 ORDER BY transitioned_at DESC;

-- ----------------------------------------------------------------------------
-- penalty_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueuePenaltyOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO penalty_outbox_events (
    penalty_id, event_type, payload, idempotency_key,
    next_attempt_at, attempts
) VALUES (
    $1, $2, $3, $4, now(), 0
)
RETURNING id, penalty_id, event_type, payload, idempotency_key,
          created_at, next_attempt_at, attempts,
          delivered_at, last_error;

-- name: LockPendingPenaltyOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, penalty_id, event_type, payload, idempotency_key,
       created_at, next_attempt_at, attempts,
       delivered_at, last_error
  FROM penalty_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkPenaltyOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE penalty_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkPenaltyOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE penalty_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
