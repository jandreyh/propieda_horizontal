-- Queries del modulo incidents (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- incidents
-- ----------------------------------------------------------------------------

-- name: CreateIncident :one
-- Crea un incidente nuevo en estado 'reported'.
INSERT INTO incidents (
    incident_type, severity, title, description,
    reported_by_user_id, reported_at,
    structure_id, location_detail,
    sla_assign_due_at, sla_resolve_due_at,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    'reported', $5, $5
)
RETURNING id, incident_type, severity, title, description,
          reported_by_user_id, reported_at,
          structure_id, location_detail,
          assigned_to_user_id, assigned_at,
          started_at, resolved_at, closed_at, cancelled_at,
          resolution_notes, escalated,
          sla_assign_due_at, sla_resolve_due_at,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetIncidentByID :one
-- Devuelve un incidente por id (no soft-deleted).
SELECT id, incident_type, severity, title, description,
       reported_by_user_id, reported_at,
       structure_id, location_detail,
       assigned_to_user_id, assigned_at,
       started_at, resolved_at, closed_at, cancelled_at,
       resolution_notes, escalated,
       sla_assign_due_at, sla_resolve_due_at,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM incidents
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListIncidents :many
-- Lista incidentes activos (no soft-deleted) ordenados por reported_at desc.
SELECT id, incident_type, severity, title, description,
       reported_by_user_id, reported_at,
       structure_id, location_detail,
       assigned_to_user_id, assigned_at,
       started_at, resolved_at, closed_at, cancelled_at,
       resolution_notes, escalated,
       sla_assign_due_at, sla_resolve_due_at,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM incidents
 WHERE deleted_at IS NULL
 ORDER BY reported_at DESC;

-- name: ListIncidentsByStatus :many
-- Lista incidentes filtrados por status.
SELECT id, incident_type, severity, title, description,
       reported_by_user_id, reported_at,
       structure_id, location_detail,
       assigned_to_user_id, assigned_at,
       started_at, resolved_at, closed_at, cancelled_at,
       resolution_notes, escalated,
       sla_assign_due_at, sla_resolve_due_at,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM incidents
 WHERE deleted_at IS NULL
   AND status = $1
 ORDER BY reported_at DESC;

-- name: ListIncidentsBySeverity :many
-- Lista incidentes filtrados por severidad.
SELECT id, incident_type, severity, title, description,
       reported_by_user_id, reported_at,
       structure_id, location_detail,
       assigned_to_user_id, assigned_at,
       started_at, resolved_at, closed_at, cancelled_at,
       resolution_notes, escalated,
       sla_assign_due_at, sla_resolve_due_at,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM incidents
 WHERE deleted_at IS NULL
   AND severity = $1
 ORDER BY reported_at DESC;

-- name: ListIncidentsByReporter :many
-- Lista incidentes reportados por un usuario.
SELECT id, incident_type, severity, title, description,
       reported_by_user_id, reported_at,
       structure_id, location_detail,
       assigned_to_user_id, assigned_at,
       started_at, resolved_at, closed_at, cancelled_at,
       resolution_notes, escalated,
       sla_assign_due_at, sla_resolve_due_at,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM incidents
 WHERE deleted_at IS NULL
   AND reported_by_user_id = $1
 ORDER BY reported_at DESC;

-- name: UpdateIncidentStatus :one
-- Actualiza el status y campos temporales con concurrencia optimista.
UPDATE incidents
   SET status              = sqlc.arg('new_status'),
       assigned_to_user_id = COALESCE(sqlc.narg('new_assigned_to_user_id'), assigned_to_user_id),
       assigned_at         = COALESCE(sqlc.narg('new_assigned_at'), assigned_at),
       started_at          = COALESCE(sqlc.narg('new_started_at'), started_at),
       resolved_at         = COALESCE(sqlc.narg('new_resolved_at'), resolved_at),
       closed_at           = COALESCE(sqlc.narg('new_closed_at'), closed_at),
       cancelled_at        = COALESCE(sqlc.narg('new_cancelled_at'), cancelled_at),
       resolution_notes    = COALESCE(sqlc.narg('new_resolution_notes'), resolution_notes),
       escalated           = COALESCE(sqlc.narg('new_escalated'), escalated),
       updated_at          = now(),
       updated_by          = sqlc.arg('updated_by'),
       version             = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, incident_type, severity, title, description,
          reported_by_user_id, reported_at,
          structure_id, location_detail,
          assigned_to_user_id, assigned_at,
          started_at, resolved_at, closed_at, cancelled_at,
          resolution_notes, escalated,
          sla_assign_due_at, sla_resolve_due_at,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- incident_attachments
-- ----------------------------------------------------------------------------

-- name: CreateIncidentAttachment :one
-- Crea un adjunto de incidente.
INSERT INTO incident_attachments (
    incident_id, url, mime_type, size_bytes,
    uploaded_by, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, 'active', $5, $5
)
RETURNING id, incident_id, url, mime_type, size_bytes,
          uploaded_by, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by;

-- name: ListAttachmentsByIncidentID :many
-- Lista adjuntos activos de un incidente.
SELECT id, incident_id, url, mime_type, size_bytes,
       uploaded_by, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM incident_attachments
 WHERE incident_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at ASC;

-- name: CountAttachmentsByIncidentID :one
-- Cuenta adjuntos activos de un incidente.
SELECT count(*)::int AS count
  FROM incident_attachments
 WHERE incident_id = $1
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- incident_status_history
-- ----------------------------------------------------------------------------

-- name: RecordIncidentStatusHistory :one
-- Inserta un registro append-only de historial de estado.
INSERT INTO incident_status_history (
    incident_id, from_status, to_status,
    transitioned_by_user_id, notes,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5,
    'recorded', $4, $4
)
RETURNING id, incident_id, from_status, to_status,
          transitioned_by_user_id, transitioned_at,
          notes, status,
          created_at, updated_at,
          created_by, updated_by;

-- name: ListStatusHistoryByIncidentID :many
-- Historial de un incidente ordenado por transitioned_at desc.
SELECT id, incident_id, from_status, to_status,
       transitioned_by_user_id, transitioned_at,
       notes, status,
       created_at, updated_at,
       created_by, updated_by
  FROM incident_status_history
 WHERE incident_id = $1
 ORDER BY transitioned_at DESC;

-- ----------------------------------------------------------------------------
-- incident_assignments
-- ----------------------------------------------------------------------------

-- name: CreateIncidentAssignment :one
-- Crea una asignacion en estado 'active'.
INSERT INTO incident_assignments (
    incident_id, assigned_to_user_id, assigned_by_user_id,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, 'active', $3, $3
)
RETURNING id, incident_id, assigned_to_user_id, assigned_by_user_id,
          assigned_at, unassigned_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by;

-- name: UnassignActiveByIncidentID :exec
-- Desactiva la asignacion activa de un incidente.
UPDATE incident_assignments
   SET status        = 'unassigned',
       unassigned_at = now(),
       updated_at    = now(),
       updated_by    = sqlc.arg('updated_by')
 WHERE incident_id = sqlc.arg('incident_id')
   AND status = 'active'
   AND deleted_at IS NULL;

-- name: GetActiveAssignmentByIncidentID :one
-- Devuelve la asignacion activa de un incidente.
SELECT id, incident_id, assigned_to_user_id, assigned_by_user_id,
       assigned_at, unassigned_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM incident_assignments
 WHERE incident_id = $1
   AND status = 'active'
   AND deleted_at IS NULL
 LIMIT 1;

-- ----------------------------------------------------------------------------
-- incident_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueueIncidentOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO incident_outbox_events (
    incident_id, event_type, payload, next_attempt_at, attempts
) VALUES (
    $1, $2, $3, now(), 0
)
RETURNING id, incident_id, event_type, payload, created_at,
          next_attempt_at, attempts, delivered_at, last_error;

-- name: LockPendingIncidentOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, incident_id, event_type, payload, created_at,
       next_attempt_at, attempts, delivered_at, last_error
  FROM incident_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkIncidentOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE incident_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkIncidentOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE incident_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
