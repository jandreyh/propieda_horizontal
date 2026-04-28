-- Tenant DB: modulo incidents (Fase 12 — incidentes y novedades de seguridad).
--
-- Crea las tablas operativas del modulo de incidentes:
--   * incidents                 : registro principal del incidente.
--   * incident_attachments      : adjuntos (foto/video) por URL.
--   * incident_status_history   : audit append-only del workflow.
--   * incident_assignments      : historial de asignaciones.
--   * incident_outbox_events    : outbox modulo-local (ADR-0005).
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version (en entidades criticas).
--   * Soft delete; los UNIQUE/INDEX usan WHERE deleted_at IS NULL.
--   * Concurrencia optimista en `incidents` via columna version.

-- ----------------------------------------------------------------------------
-- incidents
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incidents (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_type          TEXT         NOT NULL,
    severity               TEXT         NOT NULL,
    title                  TEXT         NOT NULL,
    description            TEXT         NOT NULL,
    reported_by_user_id    UUID         NOT NULL REFERENCES users(id),
    reported_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    structure_id           UUID         NULL REFERENCES residential_structures(id),
    location_detail        TEXT         NULL,
    assigned_to_user_id    UUID         NULL REFERENCES users(id),
    assigned_at            TIMESTAMPTZ  NULL,
    started_at             TIMESTAMPTZ  NULL,
    resolved_at            TIMESTAMPTZ  NULL,
    closed_at              TIMESTAMPTZ  NULL,
    cancelled_at           TIMESTAMPTZ  NULL,
    resolution_notes       TEXT         NULL,
    escalated              BOOLEAN      NOT NULL DEFAULT false,
    sla_assign_due_at      TIMESTAMPTZ  NULL,
    sla_resolve_due_at     TIMESTAMPTZ  NULL,
    status                 TEXT         NOT NULL DEFAULT 'reported',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ  NULL,
    created_by             UUID         NULL REFERENCES users(id),
    updated_by             UUID         NULL REFERENCES users(id),
    deleted_by             UUID         NULL REFERENCES users(id),
    version                INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT incidents_type_chk
        CHECK (incident_type IN ('noise', 'leak', 'damage', 'theft_attempt',
                                 'accident', 'pet_issue', 'other')),
    CONSTRAINT incidents_severity_chk
        CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT incidents_status_chk
        CHECK (status IN ('reported', 'assigned', 'in_progress',
                          'resolved', 'closed', 'cancelled')),
    CONSTRAINT incidents_close_requires_resolution_chk
        CHECK (
            status NOT IN ('resolved', 'closed')
            OR (resolution_notes IS NOT NULL AND length(btrim(resolution_notes)) > 0)
        )
);

-- Camino caliente: cola de incidentes activos por severidad.
CREATE INDEX IF NOT EXISTS incidents_status_severity_idx
    ON incidents (status, severity)
    WHERE deleted_at IS NULL;

-- Filtro "mis incidentes" (reportados por usuario).
CREATE INDEX IF NOT EXISTS incidents_reported_by_idx
    ON incidents (reported_by_user_id, reported_at DESC)
    WHERE deleted_at IS NULL;

-- Filtro "asignados a mi" (guarda).
CREATE INDEX IF NOT EXISTS incidents_assigned_to_idx
    ON incidents (assigned_to_user_id)
    WHERE deleted_at IS NULL AND assigned_to_user_id IS NOT NULL;

-- Job de escalamiento: vence SLA antes de hoy y aun no escalado.
CREATE INDEX IF NOT EXISTS incidents_sla_pending_idx
    ON incidents (sla_assign_due_at)
    WHERE deleted_at IS NULL AND escalated = false
          AND sla_assign_due_at IS NOT NULL
          AND status IN ('reported', 'assigned');

-- ----------------------------------------------------------------------------
-- incident_attachments
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incident_attachments (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id     UUID         NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
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
    CONSTRAINT incident_attachments_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT incident_attachments_size_chk
        CHECK (size_bytes >= 0)
);

CREATE INDEX IF NOT EXISTS incident_attachments_incident_idx
    ON incident_attachments (incident_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- incident_status_history (append-only)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incident_status_history (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id                 UUID         NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
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
    CONSTRAINT incident_status_history_to_status_chk
        CHECK (to_status IN ('reported', 'assigned', 'in_progress',
                             'resolved', 'closed', 'cancelled', 'escalated'))
);

CREATE INDEX IF NOT EXISTS incident_status_history_incident_idx
    ON incident_status_history (incident_id, transitioned_at DESC);

-- ----------------------------------------------------------------------------
-- incident_assignments
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incident_assignments (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id              UUID         NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    assigned_to_user_id      UUID         NOT NULL REFERENCES users(id),
    assigned_by_user_id      UUID         NOT NULL REFERENCES users(id),
    assigned_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    unassigned_at            TIMESTAMPTZ  NULL,
    status                   TEXT         NOT NULL DEFAULT 'active',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ  NULL,
    created_by               UUID         NULL REFERENCES users(id),
    updated_by               UUID         NULL REFERENCES users(id),
    deleted_by               UUID         NULL REFERENCES users(id),
    CONSTRAINT incident_assignments_status_chk
        CHECK (status IN ('active', 'unassigned'))
);

-- Solo una asignacion activa por incidente al mismo tiempo.
CREATE UNIQUE INDEX IF NOT EXISTS incident_assignments_one_active_idx
    ON incident_assignments (incident_id)
    WHERE deleted_at IS NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS incident_assignments_user_idx
    ON incident_assignments (assigned_to_user_id)
    WHERE deleted_at IS NULL AND status = 'active';

-- ----------------------------------------------------------------------------
-- incident_outbox_events (modulo-local)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incident_outbox_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id       UUID         NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    event_type        TEXT         NOT NULL,
    payload           JSONB        NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts          INTEGER      NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ  NULL,
    last_error        TEXT         NULL
);

CREATE INDEX IF NOT EXISTS incident_outbox_events_pending_idx
    ON incident_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;
