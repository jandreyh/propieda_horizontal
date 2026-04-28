-- Fase 7 — Hardening para piloto
--
-- 1. Tabla `audit_logs` inmutable (append-only) con trigger que rechaza
--    UPDATE/DELETE.
-- 2. Indices criticos sobre tablas de modulos operativos para soportar
--    las consultas mas frecuentes en produccion.
--
-- NOTA: la tabla `audit_logs` vive en cada Tenant DB. NO incluye columna
-- `tenant_id` (CLAUDE.md). El registro de eventos a nivel plataforma
-- (impersonation, lifecycle de tenants) vive en el Control Plane y se
-- migra aparte.

-- =========================================================================
-- 1. audit_logs (append-only)
-- =========================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID         NULL REFERENCES users(id),
    action        TEXT         NOT NULL,
    entity_type   TEXT         NOT NULL,
    entity_id     UUID         NULL,
    before        JSONB        NULL,
    after         JSONB        NULL,
    ip            INET         NULL,
    user_agent    TEXT         NULL,
    request_id    TEXT         NULL,
    occurred_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    severity      TEXT         NOT NULL DEFAULT 'info'
        CHECK (severity IN ('info','warn','high','critical'))
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_occurred_at
    ON audit_logs (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor
    ON audit_logs (actor_user_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity
    ON audit_logs (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action
    ON audit_logs (action);

-- Trigger anti-modificacion: rechaza cualquier UPDATE o DELETE.
-- Las correcciones se hacen como nuevos eventos, no editando el pasado.
CREATE OR REPLACE FUNCTION audit_logs_reject_modify() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only: % rejected', TG_OP
        USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
CREATE TRIGGER audit_logs_no_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_reject_modify();

DROP TRIGGER IF EXISTS audit_logs_no_delete ON audit_logs;
CREATE TRIGGER audit_logs_no_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_reject_modify();

-- =========================================================================
-- 2. Indices criticos para queries frecuentes en produccion
-- =========================================================================

-- packages: busqueda por unidad + estado (activos no archivados)
CREATE INDEX IF NOT EXISTS idx_packages_unit_status
    ON packages (unit_id, status) WHERE deleted_at IS NULL;

-- packages: pendientes de entrega para el dashboard del guarda
CREATE INDEX IF NOT EXISTS idx_packages_status_received_at
    ON packages (status, received_at) WHERE status = 'received' AND deleted_at IS NULL;

-- visitor_entries: visitas por unidad ordenadas por hora de entrada
CREATE INDEX IF NOT EXISTS idx_visitor_entries_unit_time
    ON visitor_entries (unit_id, entry_time DESC) WHERE deleted_at IS NULL;

-- user_role_assignments: lookup rapido de asignaciones activas por usuario
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_active
    ON user_role_assignments (user_id, role_id) WHERE revoked_at IS NULL AND deleted_at IS NULL;

-- announcements: feed ordenado pinned + recientes
CREATE INDEX IF NOT EXISTS idx_announcements_feed_order
    ON announcements (pinned DESC, published_at DESC)
    WHERE deleted_at IS NULL AND status = 'published';
