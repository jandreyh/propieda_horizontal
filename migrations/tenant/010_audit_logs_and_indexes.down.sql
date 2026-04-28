DROP INDEX IF EXISTS idx_announcements_feed_order;
DROP INDEX IF EXISTS idx_user_role_assignments_user_active;
DROP INDEX IF EXISTS idx_visitor_entries_unit_time;
DROP INDEX IF EXISTS idx_packages_status_received_at;
DROP INDEX IF EXISTS idx_packages_unit_status;

DROP TRIGGER IF EXISTS audit_logs_no_delete ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_reject_modify();
DROP TABLE IF EXISTS audit_logs;
