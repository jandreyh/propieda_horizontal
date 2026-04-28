-- Tenant DB: modulo parking (Fase 8 — POST-MVP).
--
-- Crea las tablas operativas del modulo de parqueaderos:
--   * parking_spaces                : espacios fisicos del conjunto.
--   * parking_assignments           : asignacion vigente o historica
--                                     de un espacio a una unidad.
--   * parking_assignment_history    : snapshot append-only de
--                                     reasignaciones para auditoria.
--   * parking_visitor_reservations  : reservas de visitantes con slot.
--   * parking_lottery_runs          : ejecuciones de sorteo (deterministas
--                                     con seed reproducible).
--   * parking_lottery_results       : resultados por unidad.
--   * parking_rules                 : overrides de reglas por tenant.
--   * parking_outbox_events         : outbox modulo-local (ADR 0005).
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version (excepto append-only
--     y outbox).
--   * Soft delete + indices con WHERE deleted_at IS NULL.
--   * UNIQUE parcial sobre parking_assignments (parking_space_id) WHERE
--     deleted_at IS NULL AND until_date IS NULL para una asignacion
--     activa por espacio.
--   * UNIQUE parcial sobre parking_visitor_reservations
--     (parking_space_id, slot_start_at) WHERE status='confirmed' AND
--     deleted_at IS NULL para no doble reserva.

-- ----------------------------------------------------------------------------
-- parking_spaces
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_spaces (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT         NOT NULL,
    type            TEXT         NOT NULL,
    structure_id    UUID         NULL REFERENCES residential_structures(id),
    level           TEXT         NULL,
    zone            TEXT         NULL,
    monthly_fee     NUMERIC(12,2) NULL,
    is_visitor      BOOLEAN      NOT NULL DEFAULT false,
    notes           TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT parking_spaces_status_chk
        CHECK (status IN ('active', 'inactive', 'maintenance', 'archived')),
    CONSTRAINT parking_spaces_type_chk
        CHECK (type IN ('covered', 'uncovered', 'motorcycle', 'bicycle',
                        'visitor', 'disabled', 'electric', 'double')),
    CONSTRAINT parking_spaces_monthly_fee_chk
        CHECK (monthly_fee IS NULL OR monthly_fee >= 0)
);

-- Codigo unico entre activos.
CREATE UNIQUE INDEX IF NOT EXISTS parking_spaces_code_unique
    ON parking_spaces (code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS parking_spaces_type_idx
    ON parking_spaces (type)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS parking_spaces_visitor_idx
    ON parking_spaces (is_visitor)
    WHERE deleted_at IS NULL AND is_visitor = true;

-- ----------------------------------------------------------------------------
-- parking_assignments
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_assignments (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    parking_space_id    UUID         NOT NULL REFERENCES parking_spaces(id) ON DELETE RESTRICT,
    unit_id             UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    vehicle_id          UUID         NULL REFERENCES vehicles(id) ON DELETE SET NULL,
    assigned_by_user_id UUID         NULL REFERENCES users(id),
    since_date          DATE         NOT NULL DEFAULT CURRENT_DATE,
    until_date          DATE         NULL,
    notes               TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT parking_assignments_status_chk
        CHECK (status IN ('active', 'closed', 'archived')),
    CONSTRAINT parking_assignments_dates_chk
        CHECK (until_date IS NULL OR until_date >= since_date)
);

-- Una asignacion activa por espacio (regla critica).
CREATE UNIQUE INDEX IF NOT EXISTS parking_assignments_space_active_unique
    ON parking_assignments (parking_space_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;

CREATE INDEX IF NOT EXISTS parking_assignments_unit_active_idx
    ON parking_assignments (unit_id)
    WHERE deleted_at IS NULL AND until_date IS NULL;

CREATE INDEX IF NOT EXISTS parking_assignments_vehicle_idx
    ON parking_assignments (vehicle_id)
    WHERE deleted_at IS NULL AND vehicle_id IS NOT NULL;

-- ----------------------------------------------------------------------------
-- parking_assignment_history (append-only)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_assignment_history (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    parking_space_id    UUID         NOT NULL REFERENCES parking_spaces(id),
    unit_id             UUID         NOT NULL REFERENCES units(id),
    assignment_id       UUID         NULL REFERENCES parking_assignments(id),
    since_date          DATE         NOT NULL,
    until_date          DATE         NULL,
    closed_reason       TEXT         NULL,
    snapshot_payload    JSONB        NULL,
    recorded_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    recorded_by         UUID         NULL REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS parking_assignment_history_space_idx
    ON parking_assignment_history (parking_space_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS parking_assignment_history_unit_idx
    ON parking_assignment_history (unit_id, recorded_at DESC);

-- ----------------------------------------------------------------------------
-- parking_visitor_reservations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_visitor_reservations (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    parking_space_id    UUID         NOT NULL REFERENCES parking_spaces(id) ON DELETE RESTRICT,
    unit_id             UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    requested_by        UUID         NOT NULL REFERENCES users(id),
    visitor_name        TEXT         NOT NULL,
    visitor_document    TEXT         NULL,
    vehicle_plate       TEXT         NULL,
    slot_start_at       TIMESTAMPTZ  NOT NULL,
    slot_end_at         TIMESTAMPTZ  NOT NULL,
    idempotency_key     TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'confirmed',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT parking_visitor_reservations_status_chk
        CHECK (status IN ('pending', 'confirmed', 'cancelled', 'no_show', 'consumed')),
    CONSTRAINT parking_visitor_reservations_slot_chk
        CHECK (slot_end_at > slot_start_at)
);

-- No doble reserva: solo 1 reserva confirmed por (espacio, slot_start_at).
CREATE UNIQUE INDEX IF NOT EXISTS parking_visitor_reservations_slot_unique
    ON parking_visitor_reservations (parking_space_id, slot_start_at)
    WHERE status = 'confirmed' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS parking_visitor_reservations_unit_idx
    ON parking_visitor_reservations (unit_id, slot_start_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS parking_visitor_reservations_active_idx
    ON parking_visitor_reservations (slot_start_at)
    WHERE deleted_at IS NULL AND status IN ('pending', 'confirmed');

CREATE UNIQUE INDEX IF NOT EXISTS parking_visitor_reservations_idem_unique
    ON parking_visitor_reservations (idempotency_key)
    WHERE idempotency_key IS NOT NULL AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- parking_lottery_runs
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_lottery_runs (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT         NOT NULL,
    seed_hash           TEXT         NOT NULL,
    criteria            JSONB        NOT NULL DEFAULT '{}'::JSONB,
    executed_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    executed_by         UUID         NOT NULL REFERENCES users(id),
    status              TEXT         NOT NULL DEFAULT 'completed',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT parking_lottery_runs_status_chk
        CHECK (status IN ('draft', 'completed', 'cancelled', 'archived'))
);

CREATE INDEX IF NOT EXISTS parking_lottery_runs_executed_idx
    ON parking_lottery_runs (executed_at DESC)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- parking_lottery_results
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_lottery_results (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    lottery_run_id      UUID         NOT NULL REFERENCES parking_lottery_runs(id) ON DELETE CASCADE,
    unit_id             UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    parking_space_id    UUID         NULL REFERENCES parking_spaces(id) ON DELETE SET NULL,
    position            INTEGER      NOT NULL,
    status              TEXT         NOT NULL DEFAULT 'allocated',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT parking_lottery_results_status_chk
        CHECK (status IN ('allocated', 'waitlist', 'declined', 'archived')),
    CONSTRAINT parking_lottery_results_position_chk
        CHECK (position > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS parking_lottery_results_run_unit_unique
    ON parking_lottery_results (lottery_run_id, unit_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS parking_lottery_results_run_idx
    ON parking_lottery_results (lottery_run_id, position);

-- ----------------------------------------------------------------------------
-- parking_rules
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_rules (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_key        TEXT         NOT NULL,
    rule_value      JSONB        NOT NULL,
    description     TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT parking_rules_status_chk
        CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS parking_rules_key_unique
    ON parking_rules (rule_key)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- parking_outbox_events
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS parking_outbox_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id      UUID         NOT NULL,
    event_type        TEXT         NOT NULL,
    payload           JSONB        NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts          INTEGER      NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ  NULL,
    last_error        TEXT         NULL
);

CREATE INDEX IF NOT EXISTS parking_outbox_events_pending_idx
    ON parking_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;
