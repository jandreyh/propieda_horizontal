-- Tenant DB: rollback del modulo packages.

DROP INDEX IF EXISTS package_outbox_events_pending_idx;
DROP TABLE IF EXISTS package_outbox_events;

DROP INDEX IF EXISTS package_delivery_events_package_idx;
DROP TABLE IF EXISTS package_delivery_events;

DROP INDEX IF EXISTS packages_received_at_idx;
DROP INDEX IF EXISTS packages_unit_status_idx;
DROP TABLE IF EXISTS packages;

DROP INDEX IF EXISTS package_categories_name_unique;
DROP TABLE IF EXISTS package_categories;
