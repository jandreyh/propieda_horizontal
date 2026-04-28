-- Tenant DB: rollback del modulo reservations (Fase 10).

DROP INDEX IF EXISTS reservations_outbox_events_pending_idx;
DROP TABLE IF EXISTS reservations_outbox_events;

DROP INDEX IF EXISTS reservation_status_history_reservation_idx;
DROP TABLE IF EXISTS reservation_status_history;

DROP INDEX IF EXISTS reservation_payments_reservation_idx;
DROP TABLE IF EXISTS reservation_payments;

DROP INDEX IF EXISTS reservations_qr_hash_idx;
DROP INDEX IF EXISTS reservations_idempotency_unique;
DROP INDEX IF EXISTS reservations_area_window_idx;
DROP INDEX IF EXISTS reservations_unit_idx;
DROP INDEX IF EXISTS reservations_slot_confirmed_unique;
DROP TABLE IF EXISTS reservations;

DROP INDEX IF EXISTS reservation_blackouts_area_window_idx;
DROP TABLE IF EXISTS reservation_blackouts;

DROP INDEX IF EXISTS common_area_rules_area_key_unique;
DROP TABLE IF EXISTS common_area_rules;

DROP INDEX IF EXISTS common_areas_code_unique;
DROP TABLE IF EXISTS common_areas;
