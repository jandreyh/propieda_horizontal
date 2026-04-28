-- Tenant DB: rollback del modulo penalties.

DROP INDEX IF EXISTS penalty_outbox_events_idempotency_idx;
DROP INDEX IF EXISTS penalty_outbox_events_pending_idx;
DROP TABLE IF EXISTS penalty_outbox_events;

DROP INDEX IF EXISTS penalty_status_history_penalty_idx;
DROP TABLE IF EXISTS penalty_status_history;

DROP INDEX IF EXISTS penalty_appeals_status_idx;
DROP INDEX IF EXISTS penalty_appeals_one_active_idx;
DROP TABLE IF EXISTS penalty_appeals;

DROP INDEX IF EXISTS penalties_recurrence_idx;
DROP INDEX IF EXISTS penalties_status_created_idx;
DROP INDEX IF EXISTS penalties_debtor_status_idx;
DROP INDEX IF EXISTS penalties_idempotency_unique;
ALTER TABLE IF EXISTS penalties DROP CONSTRAINT IF EXISTS penalties_source_incident_fk;
DROP TABLE IF EXISTS penalties;

DROP INDEX IF EXISTS penalty_catalog_code_unique;
DROP TABLE IF EXISTS penalty_catalog;
