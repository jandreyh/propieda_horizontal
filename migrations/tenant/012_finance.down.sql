-- Tenant DB: rollback del modulo finance (Fase 9).

DROP INDEX IF EXISTS finance_outbox_events_pending_idx;
DROP TABLE IF EXISTS finance_outbox_events;

DROP INDEX IF EXISTS paid_in_full_certificates_unit_idx;
DROP TABLE IF EXISTS paid_in_full_certificates;

DROP INDEX IF EXISTS period_closures_period_unique;
DROP TABLE IF EXISTS period_closures;

DROP INDEX IF EXISTS late_fee_runs_period_unique;
DROP TABLE IF EXISTS late_fee_runs;

DROP INDEX IF EXISTS payment_webhook_idempotency_received_idx;
DROP TABLE IF EXISTS payment_webhook_idempotency;

DROP INDEX IF EXISTS payment_gateway_configs_gateway_unique;
DROP TABLE IF EXISTS payment_gateway_configs;

DROP INDEX IF EXISTS accounting_entry_lines_account_idx;
DROP INDEX IF EXISTS accounting_entry_lines_entry_idx;
DROP TABLE IF EXISTS accounting_entry_lines;

DROP TRIGGER IF EXISTS tg_accounting_entries_immutable ON accounting_entries;
DROP FUNCTION IF EXISTS fn_accounting_entries_immutable();
DROP INDEX IF EXISTS accounting_entries_source_idx;
DROP INDEX IF EXISTS accounting_entries_period_idx;
DROP TABLE IF EXISTS accounting_entries;

DROP INDEX IF EXISTS payment_reversals_payment_idx;
DROP TABLE IF EXISTS payment_reversals;

DROP INDEX IF EXISTS payment_allocations_charge_idx;
DROP INDEX IF EXISTS payment_allocations_payment_idx;
DROP TABLE IF EXISTS payment_allocations;

DROP INDEX IF EXISTS payments_status_idx;
DROP INDEX IF EXISTS payments_account_idx;
DROP INDEX IF EXISTS payments_idempotency_unique;
DROP INDEX IF EXISTS payments_gateway_txn_unique;
DROP TABLE IF EXISTS payments;

DROP INDEX IF EXISTS payment_methods_code_unique;
DROP TABLE IF EXISTS payment_methods;

DROP INDEX IF EXISTS charge_items_charge_idx;
DROP TABLE IF EXISTS charge_items;

DROP INDEX IF EXISTS charges_period_idx;
DROP INDEX IF EXISTS charges_due_date_idx;
DROP INDEX IF EXISTS charges_account_status_idx;
DROP INDEX IF EXISTS charges_idempotency_unique;
DROP TABLE IF EXISTS charges;

DROP INDEX IF EXISTS billing_accounts_unit_idx;
DROP INDEX IF EXISTS billing_accounts_unit_holder_unique;
DROP TABLE IF EXISTS billing_accounts;

DROP INDEX IF EXISTS cost_centers_code_unique;
DROP TABLE IF EXISTS cost_centers;

DROP INDEX IF EXISTS chart_of_accounts_code_unique;
DROP TABLE IF EXISTS chart_of_accounts;
