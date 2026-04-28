-- Tenant DB: modulo assemblies (Fase 11 — POST-MVP).
--
-- Crea las tablas operativas del modulo de asambleas, votaciones y actas:
--   * assemblies                   : asamblea (ordinaria, extraordinaria,...).
--   * assembly_calls               : convocatorias formales.
--   * assembly_attendances         : registro de asistencia con coeficiente.
--   * assembly_proxies             : poderes registrados (apoderados).
--   * assembly_motions             : mociones/decisiones a votar.
--   * votes                        : votos individuales con hash chain.
--   * vote_evidence                : evidencia digital append-only
--                                    (prev_vote_hash + vote_hash).
--   * acts                         : actas firmables.
--   * act_signatures               : firmas presidente, secretario, testigos.
--   * assemblies_outbox_events     : outbox modulo-local.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id.
--   * Campos estandar (con version donde aplica concurrencia).
--   * Soft delete + UNIQUE/INDEX con WHERE deleted_at IS NULL.
--   * Hash chain en vote_evidence: prev_vote_hash + vote_hash.
--   * Trigger anti-UPDATE/DELETE sobre acts firmadas/archivadas.

-- ----------------------------------------------------------------------------
-- assemblies
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assemblies (
    id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT         NOT NULL,
    assembly_type           TEXT         NOT NULL,
    scheduled_at            TIMESTAMPTZ  NOT NULL,
    voting_mode             TEXT         NOT NULL DEFAULT 'coefficient',
    quorum_required_pct     NUMERIC(5,4) NOT NULL DEFAULT 0.5100,
    location                TEXT         NULL,
    notes                   TEXT         NULL,
    started_at              TIMESTAMPTZ  NULL,
    closed_at               TIMESTAMPTZ  NULL,
    status                  TEXT         NOT NULL DEFAULT 'draft',
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ  NULL,
    created_by              UUID         NULL REFERENCES users(id),
    updated_by              UUID         NULL REFERENCES users(id),
    deleted_by              UUID         NULL REFERENCES users(id),
    version                 INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT assemblies_status_chk
        CHECK (status IN ('draft', 'called', 'in_progress', 'closed',
                          'quorum_failed', 'archived')),
    CONSTRAINT assemblies_type_chk
        CHECK (assembly_type IN ('ordinaria', 'extraordinaria', 'virtual', 'mixta')),
    CONSTRAINT assemblies_voting_mode_chk
        CHECK (voting_mode IN ('coefficient', 'one_unit_one_vote')),
    CONSTRAINT assemblies_quorum_chk
        CHECK (quorum_required_pct > 0 AND quorum_required_pct <= 1)
);

CREATE INDEX IF NOT EXISTS assemblies_scheduled_idx
    ON assemblies (scheduled_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS assemblies_status_idx
    ON assemblies (status)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- assembly_calls
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assembly_calls (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assembly_id     UUID         NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    published_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    channels        JSONB        NOT NULL DEFAULT '[]'::JSONB,
    agenda          JSONB        NOT NULL DEFAULT '[]'::JSONB,
    body_md         TEXT         NULL,
    published_by    UUID         NULL REFERENCES users(id),
    status          TEXT         NOT NULL DEFAULT 'published',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT assembly_calls_status_chk
        CHECK (status IN ('draft', 'published', 'cancelled', 'archived'))
);

CREATE INDEX IF NOT EXISTS assembly_calls_assembly_idx
    ON assembly_calls (assembly_id, published_at DESC)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- assembly_attendances
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assembly_attendances (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assembly_id                 UUID         NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    unit_id                     UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    attendee_user_id            UUID         NULL REFERENCES users(id),
    represented_by_user_id      UUID         NULL REFERENCES users(id),
    coefficient_at_event        NUMERIC(7,6) NOT NULL DEFAULT 0,
    arrival_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    departure_at                TIMESTAMPTZ  NULL,
    is_remote                   BOOLEAN      NOT NULL DEFAULT false,
    has_voting_right            BOOLEAN      NOT NULL DEFAULT true,
    notes                       TEXT         NULL,
    status                      TEXT         NOT NULL DEFAULT 'present',
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ  NULL,
    created_by                  UUID         NULL REFERENCES users(id),
    updated_by                  UUID         NULL REFERENCES users(id),
    deleted_by                  UUID         NULL REFERENCES users(id),
    version                     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT assembly_attendances_status_chk
        CHECK (status IN ('present', 'left', 'voice_only', 'archived')),
    CONSTRAINT assembly_attendances_coeff_chk
        CHECK (coefficient_at_event >= 0 AND coefficient_at_event <= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS assembly_attendances_assembly_unit_unique
    ON assembly_attendances (assembly_id, unit_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS assembly_attendances_assembly_idx
    ON assembly_attendances (assembly_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- assembly_proxies (poderes)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assembly_proxies (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assembly_id         UUID         NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    grantor_user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    proxy_user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    unit_id             UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    document_url        TEXT         NULL,
    document_hash       TEXT         NULL,
    validated_at        TIMESTAMPTZ  NULL,
    validated_by        UUID         NULL REFERENCES users(id),
    revoked_at          TIMESTAMPTZ  NULL,
    status              TEXT         NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT assembly_proxies_status_chk
        CHECK (status IN ('pending', 'validated', 'rejected', 'revoked', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS assembly_proxies_unit_unique
    ON assembly_proxies (assembly_id, unit_id)
    WHERE deleted_at IS NULL AND status IN ('pending', 'validated');

CREATE INDEX IF NOT EXISTS assembly_proxies_proxy_idx
    ON assembly_proxies (assembly_id, proxy_user_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- assembly_motions
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assembly_motions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assembly_id         UUID         NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    title               TEXT         NOT NULL,
    description         TEXT         NULL,
    decision_type       TEXT         NOT NULL,
    voting_method       TEXT         NOT NULL,
    options             JSONB        NOT NULL DEFAULT '["yes","no","abstain"]'::JSONB,
    opens_at            TIMESTAMPTZ  NULL,
    closes_at           TIMESTAMPTZ  NULL,
    results             JSONB        NULL,
    status              TEXT         NOT NULL DEFAULT 'draft',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT assembly_motions_status_chk
        CHECK (status IN ('draft', 'open', 'closed', 'cancelled', 'archived')),
    CONSTRAINT assembly_motions_decision_chk
        CHECK (decision_type IN ('simple', 'qualified', 'special')),
    CONSTRAINT assembly_motions_method_chk
        CHECK (voting_method IN ('secret', 'nominal'))
);

CREATE INDEX IF NOT EXISTS assembly_motions_assembly_idx
    ON assembly_motions (assembly_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS assembly_motions_open_idx
    ON assembly_motions (assembly_id, status)
    WHERE deleted_at IS NULL AND status = 'open';

-- ----------------------------------------------------------------------------
-- votes
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS votes (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    motion_id           UUID         NOT NULL REFERENCES assembly_motions(id) ON DELETE RESTRICT,
    voter_user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    unit_id             UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    coefficient_used    NUMERIC(7,6) NOT NULL DEFAULT 0,
    option              TEXT         NOT NULL,
    cast_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    prev_vote_hash      TEXT         NULL,
    vote_hash           TEXT         NOT NULL,
    nonce               TEXT         NOT NULL,
    is_proxy_vote       BOOLEAN      NOT NULL DEFAULT false,
    status              TEXT         NOT NULL DEFAULT 'cast',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT votes_status_chk
        CHECK (status IN ('cast', 'changed', 'voided', 'archived')),
    CONSTRAINT votes_coefficient_chk
        CHECK (coefficient_used >= 0 AND coefficient_used <= 1)
);

-- Un voto activo por (motion, unit). Cambios crean nuevas filas con
-- prev_vote_hash apuntando al voto anterior, y la fila previa pasa a
-- status='changed'.
CREATE UNIQUE INDEX IF NOT EXISTS votes_motion_unit_active_unique
    ON votes (motion_id, unit_id)
    WHERE status = 'cast' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS votes_motion_idx
    ON votes (motion_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS votes_hash_unique
    ON votes (vote_hash)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- vote_evidence (append-only, hash chain)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vote_evidence (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id             UUID         NOT NULL REFERENCES votes(id) ON DELETE RESTRICT,
    motion_id           UUID         NOT NULL REFERENCES assembly_motions(id),
    prev_vote_hash      TEXT         NULL,
    vote_hash           TEXT         NOT NULL,
    payload_json        JSONB        NOT NULL,
    client_ip           INET         NULL,
    user_agent          TEXT         NULL,
    ntp_offset_ms       INTEGER      NULL,
    sealed_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS vote_evidence_vote_unique
    ON vote_evidence (vote_id);

CREATE UNIQUE INDEX IF NOT EXISTS vote_evidence_hash_unique
    ON vote_evidence (vote_hash);

CREATE INDEX IF NOT EXISTS vote_evidence_motion_chain_idx
    ON vote_evidence (motion_id, sealed_at);

CREATE INDEX IF NOT EXISTS vote_evidence_prev_hash_idx
    ON vote_evidence (prev_vote_hash)
    WHERE prev_vote_hash IS NOT NULL;

-- ----------------------------------------------------------------------------
-- acts
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS acts (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    assembly_id         UUID         NOT NULL REFERENCES assemblies(id) ON DELETE RESTRICT,
    body_md             TEXT         NOT NULL,
    pdf_url             TEXT         NULL,
    pdf_hash            TEXT         NULL,
    sealed_at           TIMESTAMPTZ  NULL,
    archive_until       DATE         NULL,
    status              TEXT         NOT NULL DEFAULT 'draft',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT acts_status_chk
        CHECK (status IN ('draft', 'signed', 'archived'))
);

CREATE INDEX IF NOT EXISTS acts_assembly_idx
    ON acts (assembly_id)
    WHERE deleted_at IS NULL;

-- Trigger inmutabilidad: una acta firmada o archivada no se puede mutar.
CREATE OR REPLACE FUNCTION fn_acts_immutable_when_signed()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF (TG_OP = 'UPDATE' AND OLD.status IN ('signed', 'archived')) THEN
        -- Permitir solo cambio a 'archived' desde 'signed' (sealing) y
        -- update de archive_until.
        IF NOT (OLD.status = 'signed' AND NEW.status = 'archived') THEN
            RAISE EXCEPTION 'acts in status % is immutable (id=%)', OLD.status, OLD.id
                USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    IF (TG_OP = 'DELETE' AND OLD.status IN ('signed', 'archived')) THEN
        RAISE EXCEPTION 'acts in status % cannot be deleted (id=%)', OLD.status, OLD.id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS tg_acts_immutable_when_signed ON acts;
CREATE TRIGGER tg_acts_immutable_when_signed
    BEFORE UPDATE OR DELETE ON acts
    FOR EACH ROW EXECUTE FUNCTION fn_acts_immutable_when_signed();

-- ----------------------------------------------------------------------------
-- act_signatures
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS act_signatures (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    act_id              UUID         NOT NULL REFERENCES acts(id) ON DELETE RESTRICT,
    signer_user_id      UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role                TEXT         NOT NULL,
    signed_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    signature_method    TEXT         NOT NULL DEFAULT 'simple_traceable',
    evidence_hash       TEXT         NOT NULL,
    client_ip           INET         NULL,
    user_agent          TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'valid',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT act_signatures_status_chk
        CHECK (status IN ('valid', 'revoked', 'archived')),
    CONSTRAINT act_signatures_role_chk
        CHECK (role IN ('president', 'secretary', 'witness', 'auditor')),
    CONSTRAINT act_signatures_method_chk
        CHECK (signature_method IN ('simple_otp', 'simple_traceable', 'pki_certified'))
);

CREATE UNIQUE INDEX IF NOT EXISTS act_signatures_act_role_unique
    ON act_signatures (act_id, role)
    WHERE deleted_at IS NULL AND status = 'valid';

CREATE INDEX IF NOT EXISTS act_signatures_act_idx
    ON act_signatures (act_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- assemblies_outbox_events
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assemblies_outbox_events (
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

CREATE INDEX IF NOT EXISTS assemblies_outbox_events_pending_idx
    ON assemblies_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;
