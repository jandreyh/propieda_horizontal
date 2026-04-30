DROP TRIGGER IF EXISTS platform_audit_logs_no_delete ON platform_audit_logs;
DROP TRIGGER IF EXISTS platform_audit_logs_no_update ON platform_audit_logs;
DROP FUNCTION IF EXISTS platform_audit_logs_immutable();
DROP TABLE IF EXISTS platform_audit_logs;
