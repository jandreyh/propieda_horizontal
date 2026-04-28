-- Tenant DB: modulo reservations (Fase 10 — POST-MVP).
--
-- Crea las tablas operativas del modulo de reservas de zonas comunes:
--   * common_areas                   : zonas comunes (salon, BBQ, gym).
--   * common_area_rules              : overrides de reglas por area.
--   * reservations                   : reservas individuales con QR.
--   * reservation_payments           : vinculo con pagos (Fase 9) o voucher.
--   * reservation_blackouts          : bloqueos por mantenimiento/asamblea.
--   * reservation_status_history     : historial de transiciones.
--   * reservations_outbox_events     : outbox modulo-local.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id.
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version (excepto outbox).
--   * Soft delete + UNIQUE/INDEX con WHERE deleted_at IS NULL.
--   * UNIQUE parcial (common_area_id, slot_start_at) WHERE
--     status='confirmed' AND deleted_at IS NULL para prevenir doble reserva.

-- ----------------------------------------------------------------------------
-- common_areas
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS common_areas (
    id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code                    TEXT         NOT NULL,
    name                    TEXT         NOT NULL,
    kind                    TEXT         NOT NULL,
    max_capacity            INTEGER      NULL,
    opening_time            TIME         NULL,
    closing_time            TIME         NULL,
    slot_duration_minutes   INTEGER      NOT NULL DEFAULT 60,
    cost_per_use            NUMERIC(12,2) NOT NULL DEFAULT 0,
    security_deposit        NUMERIC(12,2) NOT NULL DEFAULT 0,
    requires_approval       BOOLEAN      NOT NULL DEFAULT false,
    is_active               BOOLEAN      NOT NULL DEFAULT true,
    description             TEXT         NULL,
    status                  TEXT         NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ  NULL,
    created_by              UUID         NULL REFERENCES users(id),
    updated_by              UUID         NULL REFERENCES users(id),
    deleted_by              UUID         NULL REFERENCES users(id),
    version                 INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT common_areas_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT common_areas_kind_chk
        CHECK (kind IN ('salon_social', 'bbq', 'piscina', 'gym',
                        'cancha', 'sala_estudio', 'other')),
    CONSTRAINT common_areas_capacity_chk
        CHECK (max_capacity IS NULL OR max_capacity > 0),
    CONSTRAINT common_areas_slot_chk
        CHECK (slot_duration_minutes > 0),
    CONSTRAINT common_areas_amounts_chk
        CHECK (cost_per_use >= 0 AND security_deposit >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS common_areas_code_unique
    ON common_areas (code)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- common_area_rules
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS common_area_rules (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    common_area_id  UUID         NOT NULL REFERENCES common_areas(id) ON DELETE CASCADE,
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
    CONSTRAINT common_area_rules_status_chk
        CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS common_area_rules_area_key_unique
    ON common_area_rules (common_area_id, rule_key)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- reservation_blackouts
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS reservation_blackouts (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    common_area_id  UUID         NOT NULL REFERENCES common_areas(id) ON DELETE CASCADE,
    from_at         TIMESTAMPTZ  NOT NULL,
    to_at           TIMESTAMPTZ  NOT NULL,
    reason          TEXT         NOT NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT reservation_blackouts_status_chk
        CHECK (status IN ('active', 'cancelled', 'archived')),
    CONSTRAINT reservation_blackouts_dates_chk
        CHECK (to_at > from_at)
);

CREATE INDEX IF NOT EXISTS reservation_blackouts_area_window_idx
    ON reservation_blackouts (common_area_id, from_at, to_at)
    WHERE deleted_at IS NULL AND status = 'active';

-- ----------------------------------------------------------------------------
-- reservations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS reservations (
    id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    common_area_id          UUID         NOT NULL REFERENCES common_areas(id) ON DELETE RESTRICT,
    unit_id                 UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    requested_by_user_id    UUID         NOT NULL REFERENCES users(id),
    slot_start_at           TIMESTAMPTZ  NOT NULL,
    slot_end_at             TIMESTAMPTZ  NOT NULL,
    attendees_count         INTEGER      NULL,
    cost                    NUMERIC(12,2) NOT NULL DEFAULT 0,
    security_deposit        NUMERIC(12,2) NOT NULL DEFAULT 0,
    deposit_refunded        BOOLEAN      NOT NULL DEFAULT false,
    qr_code_hash            TEXT         NULL,
    idempotency_key         TEXT         NULL,
    notes                   TEXT         NULL,
    approved_by             UUID         NULL REFERENCES users(id),
    approved_at             TIMESTAMPTZ  NULL,
    cancelled_by            UUID         NULL REFERENCES users(id),
    cancelled_at            TIMESTAMPTZ  NULL,
    consumed_at             TIMESTAMPTZ  NULL,
    status                  TEXT         NOT NULL DEFAULT 'pending',
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ  NULL,
    created_by              UUID         NULL REFERENCES users(id),
    updated_by              UUID         NULL REFERENCES users(id),
    deleted_by              UUID         NULL REFERENCES users(id),
    version                 INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT reservations_status_chk
        CHECK (status IN ('pending', 'confirmed', 'cancelled', 'consumed',
                          'no_show', 'rejected', 'archived')),
    CONSTRAINT reservations_slot_chk
        CHECK (slot_end_at > slot_start_at),
    CONSTRAINT reservations_amounts_chk
        CHECK (cost >= 0 AND security_deposit >= 0),
    CONSTRAINT reservations_attendees_chk
        CHECK (attendees_count IS NULL OR attendees_count > 0)
);

-- Regla critica: no doble reserva en el mismo slot confirmado.
CREATE UNIQUE INDEX IF NOT EXISTS reservations_slot_confirmed_unique
    ON reservations (common_area_id, slot_start_at)
    WHERE status = 'confirmed' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS reservations_unit_idx
    ON reservations (unit_id, slot_start_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS reservations_area_window_idx
    ON reservations (common_area_id, slot_start_at)
    WHERE deleted_at IS NULL AND status IN ('pending', 'confirmed');

CREATE UNIQUE INDEX IF NOT EXISTS reservations_idempotency_unique
    ON reservations (idempotency_key)
    WHERE idempotency_key IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS reservations_qr_hash_idx
    ON reservations (qr_code_hash)
    WHERE qr_code_hash IS NOT NULL AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- reservation_payments
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS reservation_payments (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id      UUID         NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    payment_id          UUID         NULL,
    voucher_url         TEXT         NULL,
    amount              NUMERIC(12,2) NOT NULL,
    is_security_deposit BOOLEAN      NOT NULL DEFAULT false,
    paid_at             TIMESTAMPTZ  NULL,
    refunded_at         TIMESTAMPTZ  NULL,
    status              TEXT         NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT reservation_payments_status_chk
        CHECK (status IN ('pending', 'paid', 'refunded', 'forfeited', 'archived')),
    CONSTRAINT reservation_payments_amount_chk
        CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS reservation_payments_reservation_idx
    ON reservation_payments (reservation_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- reservation_status_history (append-only)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS reservation_status_history (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id      UUID         NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    from_status         TEXT         NULL,
    to_status           TEXT         NOT NULL,
    changed_by          UUID         NULL REFERENCES users(id),
    reason              TEXT         NULL,
    changed_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS reservation_status_history_reservation_idx
    ON reservation_status_history (reservation_id, changed_at DESC);

-- ----------------------------------------------------------------------------
-- reservations_outbox_events
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS reservations_outbox_events (
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

CREATE INDEX IF NOT EXISTS reservations_outbox_events_pending_idx
    ON reservations_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;

-- ----------------------------------------------------------------------------
-- Seed minimo de zonas comunes (catalogo arrancable, no obligatorio)
-- ----------------------------------------------------------------------------
INSERT INTO common_areas (code, name, kind, max_capacity, slot_duration_minutes)
VALUES
    ('salon_social', 'Salon Social', 'salon_social', 50, 240),
    ('bbq',          'Zona BBQ',     'bbq',          20, 240),
    ('gym',          'Gimnasio',     'gym',          15, 60),
    ('piscina',      'Piscina',      'piscina',      40, 120)
ON CONFLICT DO NOTHING;
