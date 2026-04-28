-- Tenant DB: modulo pqrs (Fase 14 — Peticiones, Quejas, Reclamos, Sugerencias).
--
-- Crea las tablas operativas:
--   * pqrs_categories         : categorizacion configurable por tenant.
--   * pqrs_tickets            : ticket principal (radicado).
--   * pqrs_responses          : respuestas y notas internas.
--   * pqrs_attachments        : adjuntos.
--   * pqrs_status_history     : audit append-only.
--   * pqrs_sla_alerts         : alertas SLA generadas por el job.
--   * pqrs_outbox_events      : outbox modulo-local.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO tenant_id.
--   * Campos estandar + soft delete.
--   * `serial_number` UNIQUE per year via UNIQUE
--     (ticket_year, serial_number) WHERE deleted_at IS NULL.

-- ----------------------------------------------------------------------------
-- pqrs_categories
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_categories (
    id                            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code                          TEXT         NOT NULL,
    name                          TEXT         NOT NULL,
    default_assignee_role_id      UUID         NULL REFERENCES roles(id),
    status                        TEXT         NOT NULL DEFAULT 'active',
    created_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                    TIMESTAMPTZ  NULL,
    created_by                    UUID         NULL REFERENCES users(id),
    updated_by                    UUID         NULL REFERENCES users(id),
    deleted_by                    UUID         NULL REFERENCES users(id),
    version                       INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pqrs_categories_status_chk
        CHECK (status IN ('active', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS pqrs_categories_code_unique
    ON pqrs_categories (code)
    WHERE deleted_at IS NULL;

-- Seed inicial: categorias estandar.
INSERT INTO pqrs_categories (code, name)
VALUES
    ('administrativo', 'Administrativo'),
    ('tecnico',        'Tecnico / mantenimiento'),
    ('financiero',     'Financiero'),
    ('convivencia',    'Convivencia')
ON CONFLICT DO NOTHING;

-- ----------------------------------------------------------------------------
-- pqrs_tickets
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_tickets (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_year              INTEGER      NOT NULL,
    serial_number            INTEGER      NOT NULL,
    pqr_type                 TEXT         NOT NULL,
    category_id              UUID         NULL REFERENCES pqrs_categories(id),
    subject                  TEXT         NOT NULL,
    body                     TEXT         NOT NULL,
    requester_user_id        UUID         NOT NULL REFERENCES users(id),
    assigned_to_user_id      UUID         NULL REFERENCES users(id),
    assigned_at              TIMESTAMPTZ  NULL,
    responded_at             TIMESTAMPTZ  NULL,
    closed_at                TIMESTAMPTZ  NULL,
    escalated_at             TIMESTAMPTZ  NULL,
    cancelled_at             TIMESTAMPTZ  NULL,
    sla_due_at               TIMESTAMPTZ  NULL,
    requester_rating         INTEGER      NULL,
    requester_feedback       TEXT         NULL,
    is_anonymous             BOOLEAN      NOT NULL DEFAULT false,
    status                   TEXT         NOT NULL DEFAULT 'radicado',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pqrs_tickets_type_chk
        CHECK (pqr_type IN ('peticion', 'queja', 'reclamo',
                            'sugerencia', 'solicitud_documental')),
    CONSTRAINT pqrs_tickets_status_chk
        CHECK (status IN ('radicado', 'en_estudio', 'respondido',
                          'cerrado', 'escalado', 'cancelado')),
    CONSTRAINT pqrs_tickets_serial_chk
        CHECK (serial_number > 0 AND ticket_year >= 2024),
    CONSTRAINT pqrs_tickets_rating_chk
        CHECK (requester_rating IS NULL
               OR (requester_rating BETWEEN 1 AND 5))
);

-- Numero de radicado UNICO por anio.
CREATE UNIQUE INDEX IF NOT EXISTS pqrs_tickets_serial_year_unique
    ON pqrs_tickets (ticket_year, serial_number)
    WHERE deleted_at IS NULL;

-- "Mis tickets" (residente).
CREATE INDEX IF NOT EXISTS pqrs_tickets_requester_idx
    ON pqrs_tickets (requester_user_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- "Asignados a mi" (responsable).
CREATE INDEX IF NOT EXISTS pqrs_tickets_assigned_idx
    ON pqrs_tickets (assigned_to_user_id, status)
    WHERE deleted_at IS NULL AND assigned_to_user_id IS NOT NULL;

-- Cola de admin por status.
CREATE INDEX IF NOT EXISTS pqrs_tickets_status_idx
    ON pqrs_tickets (status, created_at DESC)
    WHERE deleted_at IS NULL;

-- Job SLA: tickets sin responder con sla_due_at proximo.
CREATE INDEX IF NOT EXISTS pqrs_tickets_sla_idx
    ON pqrs_tickets (sla_due_at)
    WHERE deleted_at IS NULL
          AND sla_due_at IS NOT NULL
          AND status IN ('radicado', 'en_estudio');

-- ----------------------------------------------------------------------------
-- pqrs_responses
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_responses (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id                UUID         NOT NULL REFERENCES pqrs_tickets(id) ON DELETE CASCADE,
    response_type            TEXT         NOT NULL,
    body                     TEXT         NOT NULL,
    responded_by_user_id     UUID         NOT NULL REFERENCES users(id),
    responded_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    status                   TEXT         NOT NULL DEFAULT 'active',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    version                  INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT pqrs_responses_type_chk
        CHECK (response_type IN ('internal_note', 'official_response')),
    CONSTRAINT pqrs_responses_status_chk
        CHECK (status IN ('active', 'archived'))
);

CREATE INDEX IF NOT EXISTS pqrs_responses_ticket_idx
    ON pqrs_responses (ticket_id, responded_at DESC)
    WHERE deleted_at IS NULL;

-- Solo una respuesta oficial por ticket.
CREATE UNIQUE INDEX IF NOT EXISTS pqrs_responses_one_official_idx
    ON pqrs_responses (ticket_id)
    WHERE deleted_at IS NULL AND response_type = 'official_response';

-- ----------------------------------------------------------------------------
-- pqrs_attachments
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_attachments (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID         NULL REFERENCES pqrs_tickets(id) ON DELETE CASCADE,
    response_id     UUID         NULL REFERENCES pqrs_responses(id) ON DELETE CASCADE,
    url             TEXT         NOT NULL,
    mime_type       TEXT         NOT NULL,
    size_bytes      BIGINT       NOT NULL,
    uploaded_by     UUID         NOT NULL REFERENCES users(id),
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    CONSTRAINT pqrs_attachments_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT pqrs_attachments_size_chk
        CHECK (size_bytes >= 0),
    CONSTRAINT pqrs_attachments_owner_chk
        CHECK (
            (ticket_id IS NOT NULL AND response_id IS NULL) OR
            (ticket_id IS NULL AND response_id IS NOT NULL)
        )
);

CREATE INDEX IF NOT EXISTS pqrs_attachments_ticket_idx
    ON pqrs_attachments (ticket_id)
    WHERE deleted_at IS NULL AND ticket_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS pqrs_attachments_response_idx
    ON pqrs_attachments (response_id)
    WHERE deleted_at IS NULL AND response_id IS NOT NULL;

-- ----------------------------------------------------------------------------
-- pqrs_status_history (append-only)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_status_history (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id                   UUID         NOT NULL REFERENCES pqrs_tickets(id) ON DELETE CASCADE,
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
    CONSTRAINT pqrs_status_history_to_status_chk
        CHECK (to_status IN ('radicado', 'en_estudio', 'respondido',
                             'cerrado', 'escalado', 'cancelado'))
);

CREATE INDEX IF NOT EXISTS pqrs_status_history_ticket_idx
    ON pqrs_status_history (ticket_id, transitioned_at DESC);

-- ----------------------------------------------------------------------------
-- pqrs_sla_alerts
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_sla_alerts (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID         NOT NULL REFERENCES pqrs_tickets(id) ON DELETE CASCADE,
    alert_type      TEXT         NOT NULL,
    alerted_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    status          TEXT         NOT NULL DEFAULT 'emitted',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    CONSTRAINT pqrs_sla_alerts_type_chk
        CHECK (alert_type IN ('24h_warning', 'breached')),
    CONSTRAINT pqrs_sla_alerts_status_chk
        CHECK (status IN ('emitted', 'acknowledged'))
);

-- Cada tipo de alerta solo se emite una vez por ticket.
CREATE UNIQUE INDEX IF NOT EXISTS pqrs_sla_alerts_unique
    ON pqrs_sla_alerts (ticket_id, alert_type);

-- ----------------------------------------------------------------------------
-- pqrs_outbox_events (modulo-local)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pqrs_outbox_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id         UUID         NOT NULL REFERENCES pqrs_tickets(id) ON DELETE CASCADE,
    event_type        TEXT         NOT NULL,
    payload           JSONB        NOT NULL,
    idempotency_key   TEXT         NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts          INTEGER      NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ  NULL,
    last_error        TEXT         NULL
);

CREATE INDEX IF NOT EXISTS pqrs_outbox_events_pending_idx
    ON pqrs_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS pqrs_outbox_events_idempotency_idx
    ON pqrs_outbox_events (event_type, ticket_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
