-- Tenant DB: rollback del modulo notifications.

DROP INDEX IF EXISTS notification_deliveries_provider_idx;
DROP INDEX IF EXISTS notification_deliveries_outbox_idx;
DROP TABLE IF EXISTS notification_deliveries;

DROP INDEX IF EXISTS notification_outbox_recipient_idx;
DROP INDEX IF EXISTS notification_outbox_channel_status_idx;
DROP INDEX IF EXISTS notification_outbox_worker_idx;
DROP INDEX IF EXISTS notification_outbox_idempotency_unique;
DROP TABLE IF EXISTS notification_outbox;

DROP INDEX IF EXISTS notification_provider_configs_one_active_idx;
DROP INDEX IF EXISTS notification_provider_configs_unique;
DROP TABLE IF EXISTS notification_provider_configs;

DROP INDEX IF EXISTS notification_push_tokens_user_active_idx;
DROP INDEX IF EXISTS notification_push_tokens_unique;
DROP TABLE IF EXISTS notification_push_tokens;

DROP INDEX IF EXISTS notification_consents_unique;
DROP TABLE IF EXISTS notification_consents;

DROP INDEX IF EXISTS notification_preferences_user_idx;
DROP INDEX IF EXISTS notification_preferences_unique;
DROP TABLE IF EXISTS notification_preferences;

DROP INDEX IF EXISTS notification_templates_unique;
DROP TABLE IF EXISTS notification_templates;
