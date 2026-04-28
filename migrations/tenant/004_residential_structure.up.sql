-- Tenant DB: modulo residential_structure.
--
-- Modela el arbol opcional de torres / bloques / etapas / secciones del
-- conjunto residencial. La unidad habitacional concreta (apartamento)
-- vive en otro modulo (units); este modulo solo describe la jerarquia
-- estructural del edificio.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.

CREATE TABLE IF NOT EXISTS residential_structures (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT         NOT NULL,
    type         TEXT         NOT NULL,
    parent_id    UUID         NULL REFERENCES residential_structures(id),
    description  TEXT         NULL,
    order_index  INTEGER      NOT NULL DEFAULT 0,
    status       TEXT         NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ  NULL,
    created_by   UUID         NULL REFERENCES users(id),
    updated_by   UUID         NULL REFERENCES users(id),
    deleted_by   UUID         NULL REFERENCES users(id),
    version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT residential_structures_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT residential_structures_type_chk
        CHECK (type IN ('tower', 'block', 'stage', 'section', 'other'))
);

CREATE INDEX IF NOT EXISTS residential_structures_parent_idx
    ON residential_structures (parent_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS residential_structures_type_idx
    ON residential_structures (type);
