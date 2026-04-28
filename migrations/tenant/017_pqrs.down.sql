-- Tenant DB: rollback del modulo pqrs.

DROP INDEX IF EXISTS pqrs_outbox_events_idempotency_idx;
DROP INDEX IF EXISTS pqrs_outbox_events_pending_idx;
DROP TABLE IF EXISTS pqrs_outbox_events;

DROP INDEX IF EXISTS pqrs_sla_alerts_unique;
DROP TABLE IF EXISTS pqrs_sla_alerts;

DROP INDEX IF EXISTS pqrs_status_history_ticket_idx;
DROP TABLE IF EXISTS pqrs_status_history;

DROP INDEX IF EXISTS pqrs_attachments_response_idx;
DROP INDEX IF EXISTS pqrs_attachments_ticket_idx;
DROP TABLE IF EXISTS pqrs_attachments;

DROP INDEX IF EXISTS pqrs_responses_one_official_idx;
DROP INDEX IF EXISTS pqrs_responses_ticket_idx;
DROP TABLE IF EXISTS pqrs_responses;

DROP INDEX IF EXISTS pqrs_tickets_sla_idx;
DROP INDEX IF EXISTS pqrs_tickets_status_idx;
DROP INDEX IF EXISTS pqrs_tickets_assigned_idx;
DROP INDEX IF EXISTS pqrs_tickets_requester_idx;
DROP INDEX IF EXISTS pqrs_tickets_serial_year_unique;
DROP TABLE IF EXISTS pqrs_tickets;

DROP INDEX IF EXISTS pqrs_categories_code_unique;
DROP TABLE IF EXISTS pqrs_categories;
