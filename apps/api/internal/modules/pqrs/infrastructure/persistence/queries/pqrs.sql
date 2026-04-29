-- Queries del modulo pqrs (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- pqrs_categories
-- ----------------------------------------------------------------------------

-- name: CreatePQRSCategory :one
-- Crea una categoria nueva en estado 'active'.
INSERT INTO pqrs_categories (
    code, name, default_assignee_role_id, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, 'active', $4, $4
)
RETURNING id, code, name, default_assignee_role_id, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPQRSCategoryByID :one
-- Devuelve una categoria por id (no soft-deleted).
SELECT id, code, name, default_assignee_role_id, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM pqrs_categories
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListPQRSCategories :many
-- Lista categorias activas (no soft-deleted) ordenadas por code.
SELECT id, code, name, default_assignee_role_id, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM pqrs_categories
 WHERE deleted_at IS NULL
 ORDER BY code ASC;

-- name: UpdatePQRSCategory :one
-- Actualiza una categoria con concurrencia optimista.
UPDATE pqrs_categories
   SET code                     = sqlc.arg('new_code'),
       name                     = sqlc.arg('new_name'),
       default_assignee_role_id = sqlc.arg('new_default_assignee_role_id'),
       status                   = sqlc.arg('new_status'),
       updated_at               = now(),
       updated_by               = sqlc.arg('updated_by'),
       version                  = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, code, name, default_assignee_role_id, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- pqrs_tickets
-- ----------------------------------------------------------------------------

-- name: NextPQRSSerialNumber :one
-- Obtiene el siguiente serial_number para el anio dado usando advisory lock.
SELECT COALESCE(MAX(serial_number), 0) + 1 AS next_serial
  FROM pqrs_tickets
 WHERE ticket_year = $1
   AND deleted_at IS NULL;

-- name: CreatePQRSTicket :one
-- Crea un ticket nuevo en estado 'radicado'.
INSERT INTO pqrs_tickets (
    ticket_year, serial_number, pqr_type, category_id,
    subject, body, requester_user_id, is_anonymous,
    sla_due_at, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, 'radicado', $10, $10
)
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPQRSTicketByID :one
-- Devuelve un ticket por id (no soft-deleted).
SELECT id, ticket_year, serial_number, pqr_type, category_id,
       subject, body, requester_user_id,
       assigned_to_user_id, assigned_at, responded_at,
       closed_at, escalated_at, cancelled_at, sla_due_at,
       requester_rating, requester_feedback, is_anonymous,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM pqrs_tickets
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListPQRSTickets :many
-- Lista tickets filtrados y ordenados por created_at desc.
SELECT id, ticket_year, serial_number, pqr_type, category_id,
       subject, body, requester_user_id,
       assigned_to_user_id, assigned_at, responded_at,
       closed_at, escalated_at, cancelled_at, sla_due_at,
       requester_rating, requester_feedback, is_anonymous,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM pqrs_tickets
 WHERE deleted_at IS NULL
   AND (sqlc.narg('filter_status')::text IS NULL OR status = sqlc.narg('filter_status'))
   AND (sqlc.narg('filter_pqr_type')::text IS NULL OR pqr_type = sqlc.narg('filter_pqr_type'))
   AND (sqlc.narg('filter_requester')::uuid IS NULL OR requester_user_id = sqlc.narg('filter_requester'))
   AND (sqlc.narg('filter_assigned')::uuid IS NULL OR assigned_to_user_id = sqlc.narg('filter_assigned'))
 ORDER BY created_at DESC;

-- name: UpdatePQRSTicketStatus :one
-- Actualiza el status de un ticket con concurrencia optimista.
UPDATE pqrs_tickets
   SET status     = sqlc.arg('new_status'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: AssignPQRSTicket :one
-- Asigna un ticket a un usuario.
UPDATE pqrs_tickets
   SET assigned_to_user_id = sqlc.arg('assignee_user_id'),
       assigned_at         = now(),
       updated_at          = now(),
       updated_by          = sqlc.arg('updated_by'),
       version             = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: SetPQRSTicketResponded :one
-- Marca un ticket como respondido.
UPDATE pqrs_tickets
   SET status       = 'respondido',
       responded_at = now(),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ClosePQRSTicket :one
-- Cierra un ticket con rating y feedback opcionales.
UPDATE pqrs_tickets
   SET status             = 'cerrado',
       closed_at          = now(),
       requester_rating   = sqlc.narg('rating'),
       requester_feedback = sqlc.narg('feedback'),
       updated_at         = now(),
       updated_by         = sqlc.arg('updated_by'),
       version            = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: EscalatePQRSTicket :one
-- Escala un ticket.
UPDATE pqrs_tickets
   SET status       = 'escalado',
       escalated_at = now(),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: CancelPQRSTicket :one
-- Cancela un ticket.
UPDATE pqrs_tickets
   SET status       = 'cancelado',
       cancelled_at = now(),
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, ticket_year, serial_number, pqr_type, category_id,
          subject, body, requester_user_id,
          assigned_to_user_id, assigned_at, responded_at,
          closed_at, escalated_at, cancelled_at, sla_due_at,
          requester_rating, requester_feedback, is_anonymous,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- pqrs_responses
-- ----------------------------------------------------------------------------

-- name: CreatePQRSResponse :one
-- Crea una respuesta (nota interna u oficial).
INSERT INTO pqrs_responses (
    ticket_id, response_type, body,
    responded_by_user_id, responded_at,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, now(), 'active', $4, $4
)
RETURNING id, ticket_id, response_type, body,
          responded_by_user_id, responded_at,
          status, created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListPQRSResponsesByTicketID :many
-- Lista respuestas de un ticket ordenadas por responded_at desc.
SELECT id, ticket_id, response_type, body,
       responded_by_user_id, responded_at,
       status, created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM pqrs_responses
 WHERE ticket_id = $1
   AND deleted_at IS NULL
 ORDER BY responded_at DESC;

-- name: HasPQRSOfficialResponse :one
-- Indica si ya existe una respuesta oficial para el ticket.
SELECT EXISTS (
    SELECT 1
      FROM pqrs_responses
     WHERE ticket_id = $1
       AND response_type = 'official_response'
       AND deleted_at IS NULL
) AS has_official;

-- ----------------------------------------------------------------------------
-- pqrs_status_history
-- ----------------------------------------------------------------------------

-- name: RecordPQRSStatusHistory :one
-- Inserta un registro append-only de historial de transicion.
INSERT INTO pqrs_status_history (
    ticket_id, from_status, to_status,
    transitioned_by_user_id, transitioned_at,
    notes, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, now(), $5, 'recorded', $4, $4
)
RETURNING id, ticket_id, from_status, to_status,
          transitioned_by_user_id, transitioned_at,
          notes, status, created_at, updated_at,
          created_by, updated_by;

-- name: ListPQRSStatusHistoryByTicketID :many
-- Historial de un ticket ordenado por transitioned_at desc.
SELECT id, ticket_id, from_status, to_status,
       transitioned_by_user_id, transitioned_at,
       notes, status, created_at, updated_at,
       created_by, updated_by
  FROM pqrs_status_history
 WHERE ticket_id = $1
 ORDER BY transitioned_at DESC;

-- ----------------------------------------------------------------------------
-- pqrs_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueuePQRSOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO pqrs_outbox_events (
    ticket_id, event_type, payload, idempotency_key,
    next_attempt_at, attempts
) VALUES (
    $1, $2, $3, $4, now(), 0
)
RETURNING id, ticket_id, event_type, payload, idempotency_key,
          created_at, next_attempt_at, attempts,
          delivered_at, last_error;

-- name: LockPendingPQRSOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, ticket_id, event_type, payload, idempotency_key,
       created_at, next_attempt_at, attempts,
       delivered_at, last_error
  FROM pqrs_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkPQRSOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE pqrs_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkPQRSOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE pqrs_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
