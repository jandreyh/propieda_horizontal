-- Tenant DB: rollback del modulo parking (Fase 8).

DROP INDEX IF EXISTS parking_outbox_events_pending_idx;
DROP TABLE IF EXISTS parking_outbox_events;

DROP INDEX IF EXISTS parking_rules_key_unique;
DROP TABLE IF EXISTS parking_rules;

DROP INDEX IF EXISTS parking_lottery_results_run_idx;
DROP INDEX IF EXISTS parking_lottery_results_run_unit_unique;
DROP TABLE IF EXISTS parking_lottery_results;

DROP INDEX IF EXISTS parking_lottery_runs_executed_idx;
DROP TABLE IF EXISTS parking_lottery_runs;

DROP INDEX IF EXISTS parking_visitor_reservations_idem_unique;
DROP INDEX IF EXISTS parking_visitor_reservations_active_idx;
DROP INDEX IF EXISTS parking_visitor_reservations_unit_idx;
DROP INDEX IF EXISTS parking_visitor_reservations_slot_unique;
DROP TABLE IF EXISTS parking_visitor_reservations;

DROP INDEX IF EXISTS parking_assignment_history_unit_idx;
DROP INDEX IF EXISTS parking_assignment_history_space_idx;
DROP TABLE IF EXISTS parking_assignment_history;

DROP INDEX IF EXISTS parking_assignments_vehicle_idx;
DROP INDEX IF EXISTS parking_assignments_unit_active_idx;
DROP INDEX IF EXISTS parking_assignments_space_active_unique;
DROP TABLE IF EXISTS parking_assignments;

DROP INDEX IF EXISTS parking_spaces_visitor_idx;
DROP INDEX IF EXISTS parking_spaces_type_idx;
DROP INDEX IF EXISTS parking_spaces_code_unique;
DROP TABLE IF EXISTS parking_spaces;
