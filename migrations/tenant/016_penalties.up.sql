-- Tenant DB: modulo penalties (Fase 13 — multas y sanciones).
--
-- Crea las tablas operativas del modulo de sanciones:
--   * penalty_catalog           : catalogo configurable por tenant.
--   * penalties                 : sancion impuesta a un deudor.
--   * penalty_appeals           : apelaciones del residente.
--   * penalty_status_history    : audit append-only del workflow.
--   * penalty_outbox_events     : outbox modulo-local.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO tenant_id.
--   * Campos estandar + soft delete.
--   * Workflow CHECK estricto: drafted -> notified -> in_appeal ->
--     confirmed -> settled (+ dismissed, cancelled).

-- ----------------------------------------------------------------------------
-- penalty_catalog
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS penalty_catalog (
    id                            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    code                          TEXT           NOT NULL,
    name                          TEXT           NOT NULL,
    description                   TEXT           NULL,
    default_sanction_type         TEXT           NOT NULL,
    base_amount                   NUMERIC(14,2)  NOT NULL DEFAULT 0,
    recurrence_multiplier         NUMERIC(5,2)   NOT NULL DEFAULT 1.50,
    recurrence_cap_multiplier     NUMERIC(5,2)   NOT NULL DEFAULT 5.00,
    requires_council_threshold    NUMERIC(14,2)  NULL,
    status                        TEXT           NOT NULL DEFAULT 'active',
    created_at                    TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ    NOT NULL DEFAULT now(),
    deleted_at                    TIMESTAMPTZ    NULL,
    created_by                    UUID           NULL REFERENCES users(id),
    updated_by                    UUID           NULL REFERENCES users(id),
    deleted_by                    UUID           NULL REFERENCES users(id),
    version                       INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT penalty_catalog_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT penalty_catalog_default_type_chk
        CHECK (default_sanction_type IN ('warning', 'monetary', 'service_suspension')),
    CONSTRAINT penalty_catalog_amounts_chk
        CHECK (base_amount >= 0 AND recurrence_multiplier >= 1
               AND recurrence_cap_multiplier >= recurrence_multiplier)
);

CREATE UNIQUE INDEX IF NOT EXISTS penalty_catalog_code_unique
    ON penalty_catalog (code)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- penalties
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS penalties (
    id                              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id                      UUID           NOT NULL REFERENCES penalty_catalog(id) ON DELETE RESTRICT,
    debtor_user_id                  UUID           NOT NULL REFERENCES users(id),
    unit_id                         UUID           NULL REFERENCES units(id),
    source_incident_id              UUID           NULL,
    sanction_type                   TEXT           NOT NULL,
    amount                          NUMERIC(14,2)  NOT NULL DEFAULT 0,
    reason                          TEXT           NOT NULL,
    imposed_by_user_id              UUID           NOT NULL REFERENCES users(id),
    notified_at                     TIMESTAMPTZ    NULL,
    appeal_deadline_at              TIMESTAMPTZ    NULL,
    confirmed_at                    TIMESTAMPTZ    NULL,
    settled_at                      TIMESTAMPTZ    NULL,
    dismissed_at                    TIMESTAMPTZ    NULL,
    cancelled_at                    TIMESTAMPTZ    NULL,
    requires_council_approval       BOOLEAN        NOT NULL DEFAULT false,
    council_approved_by_user_id     UUID           NULL REFERENCES users(id),
    council_approved_at             TIMESTAMPTZ    NULL,
    idempotency_key                 TEXT           NULL,
    status                          TEXT           NOT NULL DEFAULT 'drafted',
    created_at                      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at                      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    deleted_at                      TIMESTAMPTZ    NULL,
    created_by                      UUID           NULL REFERENCES users(id),
    updated_by                      UUID           NULL REFERENCES users(id),
    deleted_by                      UUID           NULL REFERENCES users(id),
    version                         INTEGER        NOT NULL DEFAULT 1,
    CONSTRAINT penalties_sanction_type_chk
        CHECK (sanction_type IN ('warning', 'monetary', 'service_suspension')),
    CONSTRAINT penalties_status_chk
        CHECK (status IN ('drafted', 'notified', 'in_appeal',
                          'confirmed', 'settled', 'dismissed', 'cancelled')),
    CONSTRAINT penalties_amount_chk
        CHECK (amount >= 0),
    CONSTRAINT penalties_council_coherence_chk
        CHECK (
            requires_council_approval = false
            OR status IN ('drafted')
            OR council_approved_by_user_id IS NOT NULL
        )
);

-- FK opcional a incidents (resuelta solo si esa migracion ya corrio;
-- usamos NOT VALID para tolerar carga inicial fuera de orden).
ALTER TABLE penalties
    ADD CONSTRAINT penalties_source_incident_fk
    FOREIGN KEY (source_incident_id) REFERENCES incidents(id) NOT VALID;

-- Idempotencia de imposicion.
CREATE UNIQUE INDEX IF NOT EXISTS penalties_idempotency_unique
    ON penalties (idempotency_key)
    WHERE deleted_at IS NULL AND idempotency_key IS NOT NULL;

-- Camino caliente: sanciones del deudor.
CREATE INDEX IF NOT EXISTS penalties_debtor_status_idx
    ON penalties (debtor_user_id, status)
    WHERE deleted_at IS NULL;

-- Listado por status (cola de admin).
CREATE INDEX IF NOT EXISTS penalties_status_created_idx
    ON penalties (status, created_at DESC)
    WHERE deleted_at IS NULL;

-- Reincidencia: lookup por (debtor, catalog) en ventana temporal.
CREATE INDEX IF NOT EXISTS penalties_recurrence_idx
    ON penalties (debtor_user_id, catalog_id, confirmed_at)
    WHERE deleted_at IS NULL
          AND status IN ('confirmed', 'settled');

-- ----------------------------------------------------------------------------
-- penalty_appeals
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS penalty_appeals (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    penalty_id                  UUID         NOT NULL REFERENCES penalties(id) ON DELETE CASCADE,
    submitted_by_user_id        UUID         NOT NULL REFERENCES users(id),
    submitted_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    grounds                     TEXT         NOT NULL,
    resolved_by_user_id         UUID         NULL REFERENCES users(id),
    resolved_at                 TIMESTAMPTZ  NULL,
    resolution                  TEXT         NULL,
    status                      TEXT         NOT NULL DEFAULT 'submitted',
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ  NULL,
    created_by                  UUID         NULL REFERENCES users(id),
    updated_by                  UUID         NULL REFERENCES users(id),
    deleted_by                  UUID         NULL REFERENCES users(id),
    version                     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT penalty_appeals_status_chk
        CHECK (status IN ('submitted', 'under_review', 'accepted', 'rejected', 'withdrawn')),
    CONSTRAINT penalty_appeals_resolution_chk
        CHECK (
            status NOT IN ('accepted', 'rejected')
            OR (resolved_by_user_id IS NOT NULL
                AND resolved_at IS NOT NULL
                AND resolution IS NOT NULL
                AND length(btrim(resolution)) > 0)
        )
);

-- Una sola apelacion activa por sancion.
CREATE UNIQUE INDEX IF NOT EXISTS penalty_appeals_one_active_idx
    ON penalty_appeals (penalty_id)
    WHERE deleted_at IS NULL
          AND status IN ('submitted', 'under_review');

CREATE INDEX IF NOT EXISTS penalty_appeals_status_idx
    ON penalty_appeals (status, submitted_at DESC)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- penalty_status_history (append-only)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS penalty_status_history (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    penalty_id                  UUID         NOT NULL REFERENCES penalties(id) ON DELETE CASCADE,
    from_status                 TEXT         NULL,
    to_status                   TEXT         NOT NULL,
    transitioned_by_user_id     UUID         NOT NULL REFERENCES users(id),
    transitioned_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    notes                       TEXT         NULL,
    status                      TEXT         NOT NULL DEFAULT 'recorded',
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by                  UUID         NULL REFERENCES users(id),
    updated_by                  UUID         NULL REFERENCES users(id),
    CONSTRAINT penalty_status_history_to_status_chk
        CHECK (to_status IN ('drafted', 'notified', 'in_appeal',
                             'confirmed', 'settled', 'dismissed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS penalty_status_history_penalty_idx
    ON penalty_status_history (penalty_id, transitioned_at DESC);

-- ----------------------------------------------------------------------------
-- penalty_outbox_events (modulo-local)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS penalty_outbox_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    penalty_id        UUID         NOT NULL REFERENCES penalties(id) ON DELETE CASCADE,
    event_type        TEXT         NOT NULL,
    payload           JSONB        NOT NULL,
    idempotency_key   TEXT         NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts          INTEGER      NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ  NULL,
    last_error        TEXT         NULL
);

CREATE INDEX IF NOT EXISTS penalty_outbox_events_pending_idx
    ON penalty_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS penalty_outbox_events_idempotency_idx
    ON penalty_outbox_events (event_type, penalty_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
