-- Tenant DB: modulo units.
--
-- Crea las tres tablas operativas del modulo de unidades:
--   * units             : apartamento / casa / local / oficina / parking / storage.
--   * unit_owners       : propietarios historicos por unidad (con %).
--   * unit_occupancies  : ocupantes activos / historicos (residente, inquilino, etc.).
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * Soft delete es la regla; los UNIQUE/INDEX usan WHERE deleted_at IS NULL.
--   * structure_id apunta a residential_structures (modulo paralelo).

CREATE TABLE IF NOT EXISTS units (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    structure_id    UUID         NULL REFERENCES residential_structures(id),
    code            TEXT         NOT NULL,
    type            TEXT         NOT NULL,
    area_m2         NUMERIC(8,2) NULL,
    bedrooms        INTEGER      NULL,
    coefficient     NUMERIC(7,6) NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT units_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT units_type_chk
        CHECK (type IN ('apartment', 'house', 'commercial', 'office', 'parking', 'storage', 'other')),
    CONSTRAINT units_bedrooms_chk
        CHECK (bedrooms IS NULL OR bedrooms >= 0),
    CONSTRAINT units_area_chk
        CHECK (area_m2 IS NULL OR area_m2 >= 0),
    CONSTRAINT units_coefficient_chk
        CHECK (coefficient IS NULL OR (coefficient >= 0 AND coefficient <= 1))
);

-- Unicidad de code dentro de la torre (structure_id) cuando existe; si
-- structure_id es NULL la unicidad es por code en el conjunto. Se usa
-- COALESCE sobre el cast a texto para producir una expresion estable
-- compatible con UNIQUE INDEX parcial.
CREATE UNIQUE INDEX IF NOT EXISTS units_code_per_structure_unique
    ON units (COALESCE(structure_id::text, ''), code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS units_structure_idx
    ON units (structure_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS unit_owners (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id         UUID         NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    percentage      NUMERIC(5,2) NOT NULL DEFAULT 100.00,
    since_date      DATE         NOT NULL DEFAULT CURRENT_DATE,
    until_date      DATE         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT unit_owners_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT unit_owners_percentage_chk
        CHECK (percentage > 0 AND percentage <= 100),
    CONSTRAINT unit_owners_dates_chk
        CHECK (until_date IS NULL OR until_date >= since_date)
);

CREATE UNIQUE INDEX IF NOT EXISTS unit_owners_active_unique
    ON unit_owners (unit_id, user_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;

CREATE INDEX IF NOT EXISTS unit_owners_user_active_idx
    ON unit_owners (user_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;

CREATE TABLE IF NOT EXISTS unit_occupancies (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id         UUID         NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role_in_unit    TEXT         NOT NULL,
    is_primary      BOOLEAN      NOT NULL DEFAULT false,
    move_in_date    DATE         NOT NULL DEFAULT CURRENT_DATE,
    move_out_date   DATE         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT unit_occupancies_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT unit_occupancies_role_chk
        CHECK (role_in_unit IN ('owner_resident', 'tenant', 'authorized', 'family', 'staff')),
    CONSTRAINT unit_occupancies_dates_chk
        CHECK (move_out_date IS NULL OR move_out_date >= move_in_date)
);

CREATE INDEX IF NOT EXISTS unit_occupancies_active_idx
    ON unit_occupancies (unit_id)
    WHERE deleted_at IS NULL AND move_out_date IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS unit_occupancies_primary_unique
    ON unit_occupancies (unit_id)
    WHERE deleted_at IS NULL AND move_out_date IS NULL AND is_primary = true;
