-- Tenant DB: rollback del modulo incidents.

DROP INDEX IF EXISTS incident_outbox_events_pending_idx;
DROP TABLE IF EXISTS incident_outbox_events;

DROP INDEX IF EXISTS incident_assignments_user_idx;
DROP INDEX IF EXISTS incident_assignments_one_active_idx;
DROP TABLE IF EXISTS incident_assignments;

DROP INDEX IF EXISTS incident_status_history_incident_idx;
DROP TABLE IF EXISTS incident_status_history;

DROP INDEX IF EXISTS incident_attachments_incident_idx;
DROP TABLE IF EXISTS incident_attachments;

DROP INDEX IF EXISTS incidents_sla_pending_idx;
DROP INDEX IF EXISTS incidents_assigned_to_idx;
DROP INDEX IF EXISTS incidents_reported_by_idx;
DROP INDEX IF EXISTS incidents_status_severity_idx;
DROP TABLE IF EXISTS incidents;
