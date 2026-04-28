-- Queries del modulo packages (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista: UpdatePackageStatus usa WHERE id=$1 AND
--     version=$2; cualquier UPDATE incrementa version.
--   * Outbox modulo-local: el worker bloquea con FOR UPDATE SKIP LOCKED
--     para procesar en paralelo sin doble consumo.

-- ----------------------------------------------------------------------------
-- package_categories
-- ----------------------------------------------------------------------------

-- name: ListCategories :many
-- Devuelve las categorias activas, orden alfabetico.
SELECT id, name, requires_evidence, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM package_categories
 WHERE deleted_at IS NULL
 ORDER BY name ASC;

-- name: GetCategoryByName :one
-- Devuelve una categoria activa por nombre exacto.
SELECT id, name, requires_evidence, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM package_categories
 WHERE name = $1
   AND deleted_at IS NULL
 LIMIT 1;

-- name: GetCategoryByID :one
-- Devuelve una categoria por id (cualquier estado).
SELECT id, name, requires_evidence, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM package_categories
 WHERE id = $1
   AND deleted_at IS NULL
 LIMIT 1;

-- ----------------------------------------------------------------------------
-- packages
-- ----------------------------------------------------------------------------

-- name: CreatePackage :one
-- Crea un paquete nuevo en estado 'received'.
INSERT INTO packages (
    unit_id, recipient_name, category_id, received_evidence_url,
    carrier, tracking_number, received_by_user_id, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'received', $7, $7
)
RETURNING id, unit_id, recipient_name, category_id, received_evidence_url,
          carrier, tracking_number, received_by_user_id, received_at,
          delivered_at, returned_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPackageByID :one
-- Devuelve un paquete por id (no soft-deleted).
SELECT id, unit_id, recipient_name, category_id, received_evidence_url,
       carrier, tracking_number, received_by_user_id, received_at,
       delivered_at, returned_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM packages
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListPackagesByUnit :many
-- Lista paquetes de una unidad, ordenados por fecha de recepcion desc.
SELECT id, unit_id, recipient_name, category_id, received_evidence_url,
       carrier, tracking_number, received_by_user_id, received_at,
       delivered_at, returned_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM packages
 WHERE unit_id = $1
   AND deleted_at IS NULL
 ORDER BY received_at DESC;

-- name: ListPackagesByStatus :many
-- Lista paquetes con un status dado (camino caliente del dashboard de
-- porteria). Orden por received_at desc.
SELECT id, unit_id, recipient_name, category_id, received_evidence_url,
       carrier, tracking_number, received_by_user_id, received_at,
       delivered_at, returned_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM packages
 WHERE status = $1
   AND deleted_at IS NULL
 ORDER BY received_at DESC;

-- name: UpdatePackageStatus :one
-- Bloqueo optimista: actualiza el status si la version coincide. Si la
-- version no coincide, no afecta filas y el caller mapea a 409.
UPDATE packages
   SET status       = sqlc.arg('new_status')::TEXT,
       delivered_at = CASE
                        WHEN sqlc.arg('new_status')::TEXT = 'delivered' THEN now()
                        ELSE delivered_at
                      END,
       returned_at  = CASE
                        WHEN sqlc.arg('new_status')::TEXT = 'returned' THEN now()
                        ELSE returned_at
                      END,
       updated_at   = now(),
       updated_by   = sqlc.arg('updated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, unit_id, recipient_name, category_id, received_evidence_url,
          carrier, tracking_number, received_by_user_id, received_at,
          delivered_at, returned_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ReturnPackage :one
-- Marca el paquete como devuelto al transportador (atajo conveniente).
-- Tambien usa version optimista.
UPDATE packages
   SET status      = 'returned',
       returned_at = now(),
       updated_at  = now(),
       updated_by  = sqlc.arg('updated_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status = 'received'
RETURNING id, unit_id, recipient_name, category_id, received_evidence_url,
          carrier, tracking_number, received_by_user_id, received_at,
          delivered_at, returned_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- package_delivery_events
-- ----------------------------------------------------------------------------

-- name: RecordDeliveryEvent :one
-- Inserta el registro de la entrega del paquete (qr o manual).
INSERT INTO package_delivery_events (
    package_id, delivered_to_user_id, recipient_name_manual,
    delivery_method, signature_url, photo_evidence_url,
    delivered_by_user_id, notes, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 'completed', $7, $7
)
RETURNING id, package_id, delivered_to_user_id, recipient_name_manual,
          delivery_method, signature_url, photo_evidence_url,
          delivered_by_user_id, delivered_at, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListDeliveryEventsByPackage :many
-- Lista eventos de entrega de un paquete (por auditoria).
SELECT id, package_id, delivered_to_user_id, recipient_name_manual,
       delivery_method, signature_url, photo_evidence_url,
       delivered_by_user_id, delivered_at, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM package_delivery_events
 WHERE package_id = $1
   AND deleted_at IS NULL
 ORDER BY delivered_at DESC;

-- ----------------------------------------------------------------------------
-- package_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueueOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO package_outbox_events (
    package_id, event_type, payload, next_attempt_at, attempts
) VALUES (
    $1, $2, $3, now(), 0
)
RETURNING id, package_id, event_type, payload, created_at,
          next_attempt_at, attempts, delivered_at, last_error;

-- name: LockPendingOutboxEvents :many
-- Bloquea los eventos pendientes (delivered_at IS NULL y next_attempt_at
-- llegado) con FOR UPDATE SKIP LOCKED. Pensado para correr DENTRO de una
-- transaccion: el worker procesa el lote y commitea al final.
SELECT id, package_id, event_type, payload, created_at,
       next_attempt_at, attempts, delivered_at, last_error
  FROM package_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventDelivered :exec
-- Marca un evento como entregado (no se vuelve a procesar).
UPDATE package_outbox_events
   SET delivered_at = now(),
       attempts    = attempts + 1,
       last_error  = NULL
 WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
-- Marca un fallo: incrementa attempts, fija last_error y reagenda
-- next_attempt_at con backoff calculado por el worker (lo recibe como
-- argumento).
UPDATE package_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');

-- name: ListPackagesPendingReminder :many
-- Devuelve paquetes en estado 'received' con mas de 3 dias en porteria
-- y SIN evento 'package.reminder' encolado en las ultimas 24h. Usado por
-- el cron de re-notificacion.
SELECT p.id, p.unit_id, p.recipient_name, p.category_id, p.received_evidence_url,
       p.carrier, p.tracking_number, p.received_by_user_id, p.received_at,
       p.delivered_at, p.returned_at, p.status,
       p.created_at, p.updated_at, p.deleted_at,
       p.created_by, p.updated_by, p.deleted_by, p.version
  FROM packages p
 WHERE p.status = 'received'
   AND p.deleted_at IS NULL
   AND p.received_at < now() - INTERVAL '3 days'
   AND NOT EXISTS (
        SELECT 1
          FROM package_outbox_events e
         WHERE e.package_id = p.id
           AND e.event_type = 'package.reminder'
           AND e.created_at > now() - INTERVAL '24 hours'
   )
 ORDER BY p.received_at ASC;
