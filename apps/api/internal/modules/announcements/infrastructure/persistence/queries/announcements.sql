-- Queries del modulo announcements (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Soft-delete via marcar status='archived' y deleted_at = now().
--   * El feed por usuario ($1=user_id) usa arrays uuid[] de scopes:
--     $2 role_ids, $3 structure_ids, $4 unit_ids; arrays vacios filtran
--     correctamente porque target_id = ANY('{}'::uuid[]) es FALSE.
--   * Acknowledgments son insert ON CONFLICT DO NOTHING (idempotencia).

-- ----------------------------------------------------------------------------
-- announcements
-- ----------------------------------------------------------------------------

-- name: CreateAnnouncement :one
-- Crea un anuncio en estado 'published'. published_by_user_id se replica
-- en created_by/updated_by por defecto. published_at = now() salvo override.
INSERT INTO announcements (
    title, body, published_by_user_id, pinned, expires_at, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, 'published', $3, $3
)
RETURNING id, title, body, published_by_user_id, published_at,
          pinned, expires_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetAnnouncementByID :one
-- Devuelve un anuncio por id (no soft-deleted).
SELECT id, title, body, published_by_user_id, published_at,
       pinned, expires_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM announcements
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ArchiveAnnouncement :one
-- Soft-delete: marca el anuncio como archivado.
UPDATE announcements
   SET status     = 'archived',
       deleted_at = now(),
       deleted_by = sqlc.arg('deleted_by'),
       updated_at = now(),
       updated_by = sqlc.arg('deleted_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND deleted_at IS NULL
RETURNING id, title, body, published_by_user_id, published_at,
          pinned, expires_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListFeedForUser :many
-- Devuelve los anuncios VISIBLES para el usuario. Visible significa:
--   - status='published' AND deleted_at IS NULL,
--   - (expires_at IS NULL OR expires_at > now()),
--   - existe al menos UNA audiencia que matche:
--     target_type='global' OR
--     target_type='role' AND target_id = ANY($2::uuid[]) OR
--     target_type='structure' AND target_id = ANY($3::uuid[]) OR
--     target_type='unit' AND target_id = ANY($4::uuid[]).
-- Orden: pinned DESC, published_at DESC.
-- $1 user_id se mantiene en la firma para futuro uso (ej. excluir
-- ack-eados, segmentar por usuario), aunque hoy no participa del WHERE.
SELECT a.id, a.title, a.body, a.published_by_user_id, a.published_at,
       a.pinned, a.expires_at, a.status,
       a.created_at, a.updated_at, a.deleted_at,
       a.created_by, a.updated_by, a.deleted_by, a.version
  FROM announcements a
 WHERE a.status = 'published'
   AND a.deleted_at IS NULL
   AND (a.expires_at IS NULL OR a.expires_at > now())
   AND (sqlc.arg('user_id')::uuid IS NOT NULL OR sqlc.arg('user_id')::uuid IS NULL)
   AND EXISTS (
        SELECT 1
          FROM announcement_audiences aa
         WHERE aa.announcement_id = a.id
           AND aa.deleted_at IS NULL
           AND (
                aa.target_type = 'global'
                OR (aa.target_type = 'role'      AND aa.target_id = ANY(sqlc.arg('role_ids')::uuid[]))
                OR (aa.target_type = 'structure' AND aa.target_id = ANY(sqlc.arg('structure_ids')::uuid[]))
                OR (aa.target_type = 'unit'      AND aa.target_id = ANY(sqlc.arg('unit_ids')::uuid[]))
           )
   )
 ORDER BY a.pinned DESC, a.published_at DESC
 LIMIT sqlc.arg('lim')
 OFFSET sqlc.arg('off');

-- ----------------------------------------------------------------------------
-- announcement_audiences
-- ----------------------------------------------------------------------------

-- name: AddAudience :one
-- Inserta una audiencia para un anuncio. La coherencia
-- (target_type='global' <-> target_id NULL) se enforza por CHECK.
INSERT INTO announcement_audiences (
    announcement_id, target_type, target_id, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, 'active', $4, $4
)
RETURNING id, announcement_id, target_type, target_id, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by;

-- name: ListAudiencesByAnnouncement :many
-- Lista las audiencias activas de un anuncio.
SELECT id, announcement_id, target_type, target_id, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by
  FROM announcement_audiences
 WHERE announcement_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at ASC;

-- ----------------------------------------------------------------------------
-- announcement_acknowledgments
-- ----------------------------------------------------------------------------

-- name: Acknowledge :one
-- Inserta una confirmacion de lectura. ON CONFLICT DO NOTHING para que sea
-- idempotente (un usuario no acumula acks duplicados).
INSERT INTO announcement_acknowledgments (
    announcement_id, user_id, created_by, updated_by
) VALUES (
    $1, $2, $2, $2
)
ON CONFLICT (announcement_id, user_id) DO UPDATE
   SET acknowledged_at = announcement_acknowledgments.acknowledged_at
RETURNING id, announcement_id, user_id, acknowledged_at,
          created_at, updated_at, created_by, updated_by;
