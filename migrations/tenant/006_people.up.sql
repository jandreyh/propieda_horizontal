-- Tenant DB: modulo people.
--
-- Crea las dos tablas operativas del modulo de personas:
--   * vehicles                  : flota de vehiculos del conjunto.
--   * unit_vehicle_assignments  : asignacion historica de un vehiculo a una
--                                 unidad (apartamento). Un vehiculo puede
--                                 estar asignado a maximo una unidad activa.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * Soft delete es la regla; los UNIQUE/INDEX usan WHERE deleted_at IS NULL.

CREATE TABLE IF NOT EXISTS vehicles (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    plate        TEXT         NOT NULL,
    type         TEXT         NOT NULL,
    brand        TEXT         NULL,
    model        TEXT         NULL,
    color        TEXT         NULL,
    year         INTEGER      NULL,
    status       TEXT         NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ  NULL,
    created_by   UUID         NULL REFERENCES users(id),
    updated_by   UUID         NULL REFERENCES users(id),
    deleted_by   UUID         NULL REFERENCES users(id),
    version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT vehicles_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT vehicles_type_chk
        CHECK (type IN ('car', 'motorcycle', 'truck', 'bicycle', 'other')),
    CONSTRAINT vehicles_year_chk
        CHECK (year IS NULL OR (year >= 1950 AND year <= 2100))
);

-- Placa unica entre vehiculos no eliminados (la placa es normalizada en
-- mayusculas + trim por la capa de aplicacion antes de persistir).
CREATE UNIQUE INDEX IF NOT EXISTS vehicles_plate_unique
    ON vehicles (plate)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS unit_vehicle_assignments (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id      UUID         NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    vehicle_id   UUID         NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    since_date   DATE         NOT NULL DEFAULT CURRENT_DATE,
    until_date   DATE         NULL,
    status       TEXT         NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ  NULL,
    created_by   UUID         NULL REFERENCES users(id),
    updated_by   UUID         NULL REFERENCES users(id),
    deleted_by   UUID         NULL REFERENCES users(id),
    version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT unit_vehicle_assignments_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT unit_vehicle_assignments_dates_chk
        CHECK (until_date IS NULL OR until_date >= since_date)
);

CREATE INDEX IF NOT EXISTS unit_vehicle_assignments_unit_active_idx
    ON unit_vehicle_assignments (unit_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;

CREATE INDEX IF NOT EXISTS unit_vehicle_assignments_vehicle_active_idx
    ON unit_vehicle_assignments (vehicle_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;

-- Un vehiculo puede estar asignado activamente a una sola unidad a la vez.
CREATE UNIQUE INDEX IF NOT EXISTS unit_vehicle_assignments_vehicle_unique_active
    ON unit_vehicle_assignments (vehicle_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;
