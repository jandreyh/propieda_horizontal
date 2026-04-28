-- Tenant DB: modulo packages (paqueteria / correspondencia).
--
-- Crea las tablas operativas del modulo de paquetes:
--   * package_categories         : catalogo de categorias (sobre, caja, refrigerado).
--   * packages                   : paquetes recibidos en porteria.
--   * package_delivery_events    : eventos de entrega (qr o manual).
--   * package_outbox_events      : outbox modulo-local (ADR 0005).
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * Soft delete es la regla; los UNIQUE/INDEX usan WHERE deleted_at IS NULL.
--   * Concurrencia optimista en `packages` via columna version + UPDATE
--     ... WHERE id=$1 AND version=$2.

-- ----------------------------------------------------------------------------
-- package_categories (catalogo, sin version)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS package_categories (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name               TEXT         NOT NULL,
    requires_evidence  BOOLEAN      NOT NULL DEFAULT false,
    status             TEXT         NOT NULL DEFAULT 'active',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ  NULL,
    created_by         UUID         NULL REFERENCES users(id),
    updated_by         UUID         NULL REFERENCES users(id),
    deleted_by         UUID         NULL REFERENCES users(id),
    CONSTRAINT package_categories_status_chk
        CHECK (status IN ('active', 'archived'))
);

-- Unicidad de nombre entre categorias activas (no soft-deleted).
CREATE UNIQUE INDEX IF NOT EXISTS package_categories_name_unique
    ON package_categories (name)
    WHERE deleted_at IS NULL;

-- Seed inicial: categorias estandar del MVP.
INSERT INTO package_categories (name, requires_evidence)
VALUES
    ('Estandar',     false),
    ('Sobre',        false),
    ('Caja Grande',  false),
    ('Refrigerado', true)
ON CONFLICT DO NOTHING;

-- ----------------------------------------------------------------------------
-- packages
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS packages (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id                  UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    recipient_name           TEXT         NOT NULL,
    category_id              UUID         NULL REFERENCES package_categories(id),
    received_evidence_url    TEXT         NULL,
    carrier                  TEXT         NULL,
    tracking_number          TEXT         NULL,
    received_by_user_id      UUID         NOT NULL REFERENCES users(id),
    received_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    delivered_at             TIMESTAMPTZ  NULL,
    returned_at              TIMESTAMPTZ  NULL,
    status                   TEXT         NOT NULL DEFAULT 'received',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT packages_status_chk
        CHECK (status IN ('received', 'delivered', 'returned'))
);

-- Camino caliente: dashboard del residente y de porteria.
CREATE INDEX IF NOT EXISTS packages_unit_status_idx
    ON packages (unit_id, status)
    WHERE deleted_at IS NULL;

-- Listado por fecha (cola de paquetes pendientes / reportes).
CREATE INDEX IF NOT EXISTS packages_received_at_idx
    ON packages (received_at);

-- ----------------------------------------------------------------------------
-- package_delivery_events
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS package_delivery_events (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id               UUID         NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    delivered_to_user_id     UUID         NULL REFERENCES users(id),
    recipient_name_manual    TEXT         NULL,
    delivery_method          TEXT         NOT NULL,
    signature_url            TEXT         NULL,
    photo_evidence_url       TEXT         NULL,
    delivered_by_user_id     UUID         NOT NULL REFERENCES users(id),
    delivered_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    notes                    TEXT         NULL,
    status                   TEXT         NOT NULL DEFAULT 'completed',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT package_delivery_events_status_chk
        CHECK (status IN ('completed', 'voided')),
    CONSTRAINT package_delivery_events_method_chk
        CHECK (delivery_method IN ('qr', 'manual'))
);

CREATE INDEX IF NOT EXISTS package_delivery_events_package_idx
    ON package_delivery_events (package_id);

-- ----------------------------------------------------------------------------
-- package_outbox_events (modulo-local)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS package_outbox_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id        UUID         NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    event_type        TEXT         NOT NULL,
    payload           JSONB        NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts          INTEGER      NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ  NULL,
    last_error        TEXT         NULL
);

-- Indice del worker outbox: pendientes ordenados por proximo intento.
CREATE INDEX IF NOT EXISTS package_outbox_events_pending_idx
    ON package_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;
