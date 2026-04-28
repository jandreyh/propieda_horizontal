-- Tenant DB: modulo access_control (porteria / visitas).
--
-- Crea las tres tablas operativas del modulo de control de acceso:
--   * blacklisted_persons          : personas vetadas en porteria.
--   * visitor_pre_registrations    : pre-registros con QR firmado.
--   * visitor_entries              : entradas/salidas de visitantes.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * Soft delete es la regla; los UNIQUE/INDEX usan WHERE deleted_at IS NULL.

-- ----------------------------------------------------------------------------
-- blacklisted_persons
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS blacklisted_persons (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type         TEXT         NOT NULL,
    document_number       TEXT         NOT NULL,
    full_name             TEXT         NULL,
    reason                TEXT         NOT NULL,
    reported_by_unit_id   UUID         NULL REFERENCES units(id) ON DELETE SET NULL,
    reported_by_user_id   UUID         NULL REFERENCES users(id) ON DELETE SET NULL,
    expires_at            TIMESTAMPTZ  NULL,
    status                TEXT         NOT NULL DEFAULT 'active',
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ  NULL,
    created_by            UUID         NULL REFERENCES users(id),
    updated_by            UUID         NULL REFERENCES users(id),
    deleted_by            UUID         NULL REFERENCES users(id),
    version               INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT blacklisted_persons_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT blacklisted_persons_document_type_chk
        CHECK (document_type IN ('CC', 'CE', 'PA', 'TI', 'RC', 'NIT'))
);

-- Una sola entrada activa por (document_type, document_number).
CREATE UNIQUE INDEX IF NOT EXISTS blacklisted_persons_doc_unique
    ON blacklisted_persons (document_type, document_number)
    WHERE deleted_at IS NULL;

-- Lookup por numero de documento (camino caliente del checkin).
CREATE INDEX IF NOT EXISTS blacklisted_persons_doc_number_idx
    ON blacklisted_persons (document_number);

-- ----------------------------------------------------------------------------
-- visitor_pre_registrations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS visitor_pre_registrations (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id                  UUID         NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    created_by_user_id       UUID         NOT NULL REFERENCES users(id),
    visitor_full_name        TEXT         NOT NULL,
    visitor_document_type    TEXT         NULL,
    visitor_document_number  TEXT         NULL,
    expected_at              TIMESTAMPTZ  NULL,
    expires_at               TIMESTAMPTZ  NOT NULL,
    max_uses                 INTEGER      NOT NULL DEFAULT 1,
    uses_count               INTEGER      NOT NULL DEFAULT 0,
    qr_code_hash             TEXT         NOT NULL UNIQUE,
    status                   TEXT         NOT NULL DEFAULT 'active',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT visitor_pre_registrations_status_chk
        CHECK (status IN ('active', 'expired', 'consumed', 'revoked')),
    CONSTRAINT visitor_pre_registrations_doctype_chk
        CHECK (visitor_document_type IS NULL OR
               visitor_document_type IN ('CC', 'CE', 'PA', 'TI', 'RC', 'NIT')),
    CONSTRAINT visitor_pre_registrations_max_uses_chk
        CHECK (max_uses >= 1),
    CONSTRAINT visitor_pre_registrations_uses_count_chk
        CHECK (uses_count >= 0)
);

-- Lookup por unidad de pre-registros activos.
CREATE INDEX IF NOT EXISTS visitor_pre_registrations_unit_active_idx
    ON visitor_pre_registrations (unit_id)
    WHERE deleted_at IS NULL AND status = 'active';

-- ----------------------------------------------------------------------------
-- visitor_entries
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS visitor_entries (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id                  UUID         NULL REFERENCES units(id) ON DELETE SET NULL,
    pre_registration_id      UUID         NULL REFERENCES visitor_pre_registrations(id) ON DELETE SET NULL,
    visitor_full_name        TEXT         NOT NULL,
    visitor_document_type    TEXT         NULL,
    visitor_document_number  TEXT         NOT NULL,
    photo_url                TEXT         NULL,
    guard_id                 UUID         NOT NULL REFERENCES users(id),
    entry_time               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    exit_time                TIMESTAMPTZ  NULL,
    source                   TEXT         NOT NULL,
    notes                    TEXT         NULL,
    status                   TEXT         NOT NULL DEFAULT 'active',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT visitor_entries_status_chk
        CHECK (status IN ('active', 'closed', 'rejected')),
    CONSTRAINT visitor_entries_source_chk
        CHECK (source IN ('qr', 'manual')),
    CONSTRAINT visitor_entries_doctype_chk
        CHECK (visitor_document_type IS NULL OR
               visitor_document_type IN ('CC', 'CE', 'PA', 'TI', 'RC', 'NIT'))
);

-- Visitas activas por unidad (sin exit_time): camino caliente del dashboard.
CREATE INDEX IF NOT EXISTS visitor_entries_unit_active_idx
    ON visitor_entries (unit_id)
    WHERE exit_time IS NULL AND deleted_at IS NULL;

-- Listados ordenados por tiempo de entrada (descendente).
CREATE INDEX IF NOT EXISTS visitor_entries_entry_time_idx
    ON visitor_entries (entry_time DESC);

-- Lookup por documento del visitante (auditoria / blacklist hits).
CREATE INDEX IF NOT EXISTS visitor_entries_doc_number_idx
    ON visitor_entries (visitor_document_number);
