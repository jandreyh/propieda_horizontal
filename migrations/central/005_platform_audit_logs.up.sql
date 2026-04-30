-- Auditoria de eventos de plataforma (no de operacion del tenant).
-- ADR 0007 seccion 9: provisioning, impersonation, suspension de tenants
-- y usuarios. Los logs operativos siguen viviendo en cada tenant DB.

CREATE TABLE IF NOT EXISTS platform_audit_logs (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id   UUID         NULL REFERENCES platform_users(id) ON DELETE SET NULL,
    action          TEXT         NOT NULL,
    target_type     TEXT         NULL,
    target_id       UUID         NULL,
    metadata        JSONB        NULL,
    ip              INET         NULL,
    user_agent      TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Append-only: rechaza UPDATE/DELETE para preservar trazabilidad.
CREATE OR REPLACE FUNCTION platform_audit_logs_immutable()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'platform_audit_logs es append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS platform_audit_logs_no_update ON platform_audit_logs;
CREATE TRIGGER platform_audit_logs_no_update
    BEFORE UPDATE ON platform_audit_logs
    FOR EACH ROW EXECUTE FUNCTION platform_audit_logs_immutable();

DROP TRIGGER IF EXISTS platform_audit_logs_no_delete ON platform_audit_logs;
CREATE TRIGGER platform_audit_logs_no_delete
    BEFORE DELETE ON platform_audit_logs
    FOR EACH ROW EXECUTE FUNCTION platform_audit_logs_immutable();

CREATE INDEX IF NOT EXISTS platform_audit_logs_actor_idx
    ON platform_audit_logs (actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS platform_audit_logs_target_idx
    ON platform_audit_logs (target_type, target_id);

CREATE INDEX IF NOT EXISTS platform_audit_logs_action_idx
    ON platform_audit_logs (action, created_at DESC);
