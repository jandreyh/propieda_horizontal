-- Tenant DB: modulo announcements (tablero de anuncios).
--
-- Crea las tablas operativas del modulo de anuncios:
--   * announcements                : anuncios publicados por staff/admin.
--   * announcement_audiences       : audiencias destinatarias del anuncio.
--   * announcement_acknowledgments : confirmaciones de lectura por usuario.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * Soft delete es la regla; los UNIQUE/INDEX usan WHERE deleted_at IS NULL.

-- ----------------------------------------------------------------------------
-- announcements
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS announcements (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title                    TEXT         NOT NULL,
    body                     TEXT         NOT NULL,
    published_by_user_id     UUID         NOT NULL REFERENCES users(id),
    published_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    pinned                   BOOLEAN      NOT NULL DEFAULT false,
    expires_at               TIMESTAMPTZ  NULL,
    status                   TEXT         NOT NULL DEFAULT 'published',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT announcements_status_chk
        CHECK (status IN ('published', 'archived'))
);

-- Camino caliente del feed: anuncios mas recientes primero.
CREATE INDEX IF NOT EXISTS announcements_published_at_idx
    ON announcements (published_at DESC)
    WHERE deleted_at IS NULL;

-- Job de expiracion: localiza anuncios con expires_at proximo.
CREATE INDEX IF NOT EXISTS announcements_expires_at_idx
    ON announcements (expires_at)
    WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- Pinned: filtro frecuente en el feed (poca cardinalidad).
CREATE INDEX IF NOT EXISTS announcements_pinned_idx
    ON announcements (pinned)
    WHERE deleted_at IS NULL AND pinned = true;

-- ----------------------------------------------------------------------------
-- announcement_audiences
-- ----------------------------------------------------------------------------
-- Sin columna `version` (registro auxiliar de relacion).
CREATE TABLE IF NOT EXISTS announcement_audiences (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    announcement_id   UUID         NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    target_type       TEXT         NOT NULL,
    target_id         UUID         NULL,
    status            TEXT         NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ  NULL,
    created_by        UUID         NULL REFERENCES users(id),
    updated_by        UUID         NULL REFERENCES users(id),
    deleted_by        UUID         NULL REFERENCES users(id),
    CONSTRAINT announcement_audiences_target_type_chk
        CHECK (target_type IN ('global', 'structure', 'role', 'unit')),
    CONSTRAINT announcement_audiences_target_coherence_chk
        CHECK (
            (target_type = 'global' AND target_id IS NULL) OR
            (target_type <> 'global' AND target_id IS NOT NULL)
        ),
    CONSTRAINT announcement_audiences_status_chk
        CHECK (status IN ('active', 'archived'))
);

-- Lookup por anuncio (cargar sus audiencias).
CREATE INDEX IF NOT EXISTS announcement_audiences_announcement_idx
    ON announcement_audiences (announcement_id)
    WHERE deleted_at IS NULL;

-- Lookup por (target_type, target_id) — feed por usuario.
CREATE INDEX IF NOT EXISTS announcement_audiences_target_idx
    ON announcement_audiences (target_type, target_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- announcement_acknowledgments
-- ----------------------------------------------------------------------------
-- Sin `version` y sin `deleted_*`: registro append-only de confirmaciones.
CREATE TABLE IF NOT EXISTS announcement_acknowledgments (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    announcement_id   UUID         NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    user_id           UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    acknowledged_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by        UUID         NULL REFERENCES users(id),
    updated_by        UUID         NULL REFERENCES users(id),
    CONSTRAINT announcement_acknowledgments_unique
        UNIQUE (announcement_id, user_id)
);

-- Lookup por usuario (mis confirmaciones).
CREATE INDEX IF NOT EXISTS announcement_acknowledgments_user_idx
    ON announcement_acknowledgments (user_id);
