-- Queries del modulo access_control (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Soft-delete via marcar status='archived' y deleted_at = now().
--   * Concurrencia optimista: cada update incrementa version.
--   * Las queries de pre-registro consideran "expirado" cuando expires_at
--     es NULL o ya paso. La capa de aplicacion entrega el QR plano una
--     unica vez; aqui solo guardamos el hash sha256.

-- ----------------------------------------------------------------------------
-- blacklisted_persons
-- ----------------------------------------------------------------------------

-- name: GetBlacklistByDocument :one
-- Devuelve la entrada activa de blacklist para (document_type, document_number)
-- si existe y no esta expirada. Camino caliente del checkin.
SELECT id, document_type, document_number, full_name, reason,
       reported_by_unit_id, reported_by_user_id, expires_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM blacklisted_persons
 WHERE document_type = $1
   AND document_number = $2
   AND deleted_at IS NULL
   AND (expires_at IS NULL OR expires_at > now())
 LIMIT 1;

-- name: CreateBlacklistEntry :one
-- Crea una entrada nueva en la blacklist.
INSERT INTO blacklisted_persons (
    document_type, document_number, full_name, reason,
    reported_by_unit_id, reported_by_user_id, expires_at, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'active', $8, $8
)
RETURNING id, document_type, document_number, full_name, reason,
          reported_by_unit_id, reported_by_user_id, expires_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListBlacklist :many
-- Lista las entradas activas de blacklist (no eliminadas) ordenadas por
-- created_at desc.
SELECT id, document_type, document_number, full_name, reason,
       reported_by_unit_id, reported_by_user_id, expires_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM blacklisted_persons
 WHERE deleted_at IS NULL
 ORDER BY created_at DESC;

-- name: ArchiveBlacklistEntry :one
-- Soft-delete: marca la entrada como archivada.
UPDATE blacklisted_persons
   SET status     = 'archived',
       deleted_at = now(),
       deleted_by = sqlc.arg('deleted_by'),
       updated_at = now(),
       updated_by = sqlc.arg('deleted_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND deleted_at IS NULL
RETURNING id, document_type, document_number, full_name, reason,
          reported_by_unit_id, reported_by_user_id, expires_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- visitor_pre_registrations
-- ----------------------------------------------------------------------------

-- name: CreatePreRegistration :one
-- Crea un pre-registro nuevo. El qr_code_hash es el sha256 del codigo plano
-- generado por la capa de aplicacion (que entrega el plano UNA vez al
-- cliente).
INSERT INTO visitor_pre_registrations (
    unit_id, created_by_user_id, visitor_full_name,
    visitor_document_type, visitor_document_number, expected_at,
    expires_at, max_uses, uses_count, qr_code_hash, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 0, $9, 'active', $2, $2
)
RETURNING id, unit_id, created_by_user_id, visitor_full_name,
          visitor_document_type, visitor_document_number, expected_at,
          expires_at, max_uses, uses_count, qr_code_hash, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPreRegistrationByQRHash :one
-- Devuelve el pre-registro por hash del QR (no aplica filtro de estado;
-- el caller decide).
SELECT id, unit_id, created_by_user_id, visitor_full_name,
       visitor_document_type, visitor_document_number, expected_at,
       expires_at, max_uses, uses_count, qr_code_hash, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM visitor_pre_registrations
 WHERE qr_code_hash = $1
   AND deleted_at IS NULL;

-- name: IncrementPreRegistrationUses :one
-- Incrementa uses_count atomicamente si el pre-registro:
--   * sigue activo,
--   * no expiro,
--   * todavia tiene cupos disponibles.
-- Si se alcanza max_uses, marca como 'consumed'. Si no se cumple alguna
-- condicion, devuelve 0 filas (el caller mapea a 410 Gone).
UPDATE visitor_pre_registrations
   SET uses_count = uses_count + 1,
       status = CASE
                  WHEN uses_count + 1 >= max_uses THEN 'consumed'
                  ELSE status
                END,
       updated_at = now(),
       version = version + 1
 WHERE qr_code_hash = $1
   AND status = 'active'
   AND deleted_at IS NULL
   AND expires_at > now()
   AND uses_count < max_uses
RETURNING id, unit_id, created_by_user_id, visitor_full_name,
          visitor_document_type, visitor_document_number, expected_at,
          expires_at, max_uses, uses_count, qr_code_hash, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- visitor_entries
-- ----------------------------------------------------------------------------

-- name: CreateVisitorEntry :one
-- Inserta una entrada de visitante. La capa de aplicacion decide el status
-- ('active' para checkin valido, 'rejected' para intento bloqueado por
-- blacklist).
INSERT INTO visitor_entries (
    unit_id, pre_registration_id, visitor_full_name,
    visitor_document_type, visitor_document_number, photo_url,
    guard_id, source, notes, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $7, $7
)
RETURNING id, unit_id, pre_registration_id, visitor_full_name,
          visitor_document_type, visitor_document_number, photo_url,
          guard_id, entry_time, exit_time, source, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: CloseVisitorEntry :one
-- Cierra una entrada activa fijando exit_time = now() y status='closed'.
UPDATE visitor_entries
   SET exit_time = now(),
       status    = 'closed',
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version   = version + 1
 WHERE id = sqlc.arg('id')
   AND deleted_at IS NULL
   AND exit_time IS NULL
   AND status = 'active'
RETURNING id, unit_id, pre_registration_id, visitor_full_name,
          visitor_document_type, visitor_document_number, photo_url,
          guard_id, entry_time, exit_time, source, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListActiveVisits :many
-- Visitas activas: status='active', sin exit_time. Para el dashboard del
-- guarda.
SELECT id, unit_id, pre_registration_id, visitor_full_name,
       visitor_document_type, visitor_document_number, photo_url,
       guard_id, entry_time, exit_time, source, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM visitor_entries
 WHERE deleted_at IS NULL
   AND exit_time IS NULL
   AND status = 'active'
 ORDER BY entry_time DESC;

-- name: GetEntryByID :one
-- Devuelve una entrada por id (cualquier estado, no eliminada).
SELECT id, unit_id, pre_registration_id, visitor_full_name,
       visitor_document_type, visitor_document_number, photo_url,
       guard_id, entry_time, exit_time, source, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM visitor_entries
 WHERE id = $1
   AND deleted_at IS NULL;
