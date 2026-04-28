-- Tenant DB: modulo finance (Fase 9 — POST-MVP).
--
-- Crea las tablas operativas del modulo financiero:
--   * chart_of_accounts            : plan de cuentas por tenant.
--   * cost_centers                 : centros de costo.
--   * billing_accounts             : cuenta contrato unidad+titular.
--   * charges                      : cargos (cuotas, multas, intereses).
--   * charge_items                 : detalle linea por cargo.
--   * payment_methods              : catalogo de metodos.
--   * payments                     : pagos manual o pasarela. UNIQUE
--                                    parcial sobre gateway_txn_id.
--   * payment_allocations          : aplicacion de pagos a cargos.
--   * payment_reversals            : reversos con doble validacion.
--   * accounting_entries           : asientos contables. Sealed via
--                                    cierre hard => trigger inmutabilidad.
--   * accounting_entry_lines       : lineas de cada asiento.
--   * payment_gateway_configs      : credenciales por tenant/gateway.
--   * payment_webhook_idempotency  : dedup de webhooks. UNIQUE
--                                    (gateway, idempotency_key).
--   * late_fee_runs                : ejecucion idempotente de intereses.
--   * period_closures              : cierres soft/hard mensual y anual.
--   * paid_in_full_certificates    : certificados de paz y salvo.
--   * finance_outbox_events        : outbox modulo-local.
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id.
--   * Campos estandar (con version donde aplica concurrencia).
--   * Soft delete + indices con WHERE deleted_at IS NULL.
--   * UNIQUE(gateway, idempotency_key) en payment_webhook_idempotency.
--   * UNIQUE(gateway_txn_id) parcial WHERE NOT NULL en payments.
--   * Trigger inmutabilidad sobre accounting_entries con sealed=true.

-- ----------------------------------------------------------------------------
-- chart_of_accounts
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS chart_of_accounts (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT         NOT NULL,
    name            TEXT         NOT NULL,
    account_type    TEXT         NOT NULL,
    parent_id       UUID         NULL REFERENCES chart_of_accounts(id) ON DELETE RESTRICT,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT chart_of_accounts_status_chk
        CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT chart_of_accounts_type_chk
        CHECK (account_type IN ('asset', 'liability', 'equity', 'income', 'expense'))
);

CREATE UNIQUE INDEX IF NOT EXISTS chart_of_accounts_code_unique
    ON chart_of_accounts (code)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- cost_centers
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cost_centers (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT         NOT NULL,
    name            TEXT         NOT NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT cost_centers_status_chk
        CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS cost_centers_code_unique
    ON cost_centers (code)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- billing_accounts
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS billing_accounts (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id         UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    holder_user_id  UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    opened_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    closed_at       TIMESTAMPTZ  NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT billing_accounts_status_chk
        CHECK (status IN ('active', 'closed', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS billing_accounts_unit_holder_unique
    ON billing_accounts (unit_id, holder_user_id)
    WHERE deleted_at IS NULL AND closed_at IS NULL;

CREATE INDEX IF NOT EXISTS billing_accounts_unit_idx
    ON billing_accounts (unit_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- charges
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS charges (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id  UUID         NOT NULL REFERENCES billing_accounts(id) ON DELETE RESTRICT,
    concept             TEXT         NOT NULL,
    period_year         INTEGER      NULL,
    period_month        INTEGER      NULL,
    amount              NUMERIC(14,2) NOT NULL,
    balance             NUMERIC(14,2) NOT NULL,
    due_date            DATE         NOT NULL,
    cost_center_id      UUID         NULL REFERENCES cost_centers(id),
    account_id          UUID         NULL REFERENCES chart_of_accounts(id),
    idempotency_key     TEXT         NULL,
    description         TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'open',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT charges_concept_chk
        CHECK (concept IN ('admin_fee', 'late_fee', 'interest', 'service',
                           'rental', 'penalty', 'other')),
    CONSTRAINT charges_status_chk
        CHECK (status IN ('open', 'partial', 'paid', 'voided', 'archived')),
    CONSTRAINT charges_amount_chk
        CHECK (amount >= 0),
    CONSTRAINT charges_balance_chk
        CHECK (balance >= 0),
    CONSTRAINT charges_period_month_chk
        CHECK (period_month IS NULL OR (period_month >= 1 AND period_month <= 12))
);

CREATE UNIQUE INDEX IF NOT EXISTS charges_idempotency_unique
    ON charges (idempotency_key)
    WHERE idempotency_key IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS charges_account_status_idx
    ON charges (billing_account_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS charges_due_date_idx
    ON charges (due_date)
    WHERE deleted_at IS NULL AND status IN ('open', 'partial');

CREATE INDEX IF NOT EXISTS charges_period_idx
    ON charges (period_year, period_month)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- charge_items
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS charge_items (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    charge_id       UUID         NOT NULL REFERENCES charges(id) ON DELETE CASCADE,
    description     TEXT         NOT NULL,
    amount          NUMERIC(14,2) NOT NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT charge_items_status_chk
        CHECK (status IN ('active', 'voided', 'archived')),
    CONSTRAINT charge_items_amount_chk
        CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS charge_items_charge_idx
    ON charge_items (charge_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- payment_methods (catalogo)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_methods (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT         NOT NULL,
    name            TEXT         NOT NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT payment_methods_status_chk
        CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS payment_methods_code_unique
    ON payment_methods (code)
    WHERE deleted_at IS NULL;

INSERT INTO payment_methods (code, name) VALUES
    ('cash',           'Efectivo'),
    ('bank_transfer',  'Transferencia bancaria'),
    ('pse',            'PSE'),
    ('credit_card',    'Tarjeta credito'),
    ('debit_card',     'Tarjeta debito'),
    ('voucher',        'Comprobante manual')
ON CONFLICT DO NOTHING;

-- ----------------------------------------------------------------------------
-- payments
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payments (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id  UUID         NOT NULL REFERENCES billing_accounts(id) ON DELETE RESTRICT,
    payer_user_id       UUID         NULL REFERENCES users(id),
    method_code         TEXT         NOT NULL,
    gateway             TEXT         NULL,
    gateway_txn_id      TEXT         NULL,
    idempotency_key     TEXT         NULL,
    amount              NUMERIC(14,2) NOT NULL,
    currency            TEXT         NOT NULL DEFAULT 'COP',
    unallocated_amount  NUMERIC(14,2) NOT NULL DEFAULT 0,
    captured_at         TIMESTAMPTZ  NULL,
    settled_at          TIMESTAMPTZ  NULL,
    failure_reason      TEXT         NULL,
    receipt_number      TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT payments_status_chk
        CHECK (status IN ('pending', 'authorized', 'captured', 'settled',
                          'reversed', 'failed', 'archived')),
    CONSTRAINT payments_amount_chk
        CHECK (amount > 0),
    CONSTRAINT payments_unallocated_chk
        CHECK (unallocated_amount >= 0 AND unallocated_amount <= amount),
    CONSTRAINT payments_currency_chk
        CHECK (length(currency) = 3)
);

-- Doble pago en pasarela bloqueado: UNIQUE parcial.
CREATE UNIQUE INDEX IF NOT EXISTS payments_gateway_txn_unique
    ON payments (gateway_txn_id)
    WHERE gateway_txn_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS payments_idempotency_unique
    ON payments (idempotency_key)
    WHERE idempotency_key IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS payments_account_idx
    ON payments (billing_account_id, captured_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS payments_status_idx
    ON payments (status)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- payment_allocations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_allocations (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id      UUID         NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    charge_id       UUID         NOT NULL REFERENCES charges(id) ON DELETE RESTRICT,
    amount          NUMERIC(14,2) NOT NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT payment_allocations_status_chk
        CHECK (status IN ('active', 'reversed', 'archived')),
    CONSTRAINT payment_allocations_amount_chk
        CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS payment_allocations_payment_idx
    ON payment_allocations (payment_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS payment_allocations_charge_idx
    ON payment_allocations (charge_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- payment_reversals
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_reversals (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id      UUID         NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    reason          TEXT         NOT NULL,
    requested_by    UUID         NOT NULL REFERENCES users(id),
    requested_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    approved_by     UUID         NULL REFERENCES users(id),
    approved_at     TIMESTAMPTZ  NULL,
    completed_at    TIMESTAMPTZ  NULL,
    status          TEXT         NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT payment_reversals_status_chk
        CHECK (status IN ('pending', 'approved', 'rejected', 'completed', 'archived'))
);

CREATE INDEX IF NOT EXISTS payment_reversals_payment_idx
    ON payment_reversals (payment_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- accounting_entries
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounting_entries (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    period_year     INTEGER      NOT NULL,
    period_month    INTEGER      NOT NULL,
    posted_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    source_type     TEXT         NOT NULL,
    source_id       UUID         NOT NULL,
    description     TEXT         NULL,
    posted          BOOLEAN      NOT NULL DEFAULT true,
    sealed          BOOLEAN      NOT NULL DEFAULT false,
    sealed_at       TIMESTAMPTZ  NULL,
    status          TEXT         NOT NULL DEFAULT 'posted',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT accounting_entries_status_chk
        CHECK (status IN ('draft', 'posted', 'reversed', 'archived')),
    CONSTRAINT accounting_entries_period_month_chk
        CHECK (period_month >= 1 AND period_month <= 12)
);

CREATE INDEX IF NOT EXISTS accounting_entries_period_idx
    ON accounting_entries (period_year, period_month)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS accounting_entries_source_idx
    ON accounting_entries (source_type, source_id)
    WHERE deleted_at IS NULL;

-- Trigger inmutabilidad: rechaza UPDATE/DELETE sobre entries selladas.
CREATE OR REPLACE FUNCTION fn_accounting_entries_immutable()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF (TG_OP = 'UPDATE' AND OLD.sealed = true) THEN
        RAISE EXCEPTION 'accounting_entries sealed=true is immutable (id=%)', OLD.id
            USING ERRCODE = 'check_violation';
    END IF;
    IF (TG_OP = 'DELETE' AND OLD.sealed = true) THEN
        RAISE EXCEPTION 'accounting_entries sealed=true cannot be deleted (id=%)', OLD.id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS tg_accounting_entries_immutable ON accounting_entries;
CREATE TRIGGER tg_accounting_entries_immutable
    BEFORE UPDATE OR DELETE ON accounting_entries
    FOR EACH ROW EXECUTE FUNCTION fn_accounting_entries_immutable();

-- ----------------------------------------------------------------------------
-- accounting_entry_lines
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounting_entry_lines (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id        UUID         NOT NULL REFERENCES accounting_entries(id) ON DELETE RESTRICT,
    account_id      UUID         NOT NULL REFERENCES chart_of_accounts(id) ON DELETE RESTRICT,
    cost_center_id  UUID         NULL REFERENCES cost_centers(id),
    debit           NUMERIC(14,2) NOT NULL DEFAULT 0,
    credit          NUMERIC(14,2) NOT NULL DEFAULT 0,
    description     TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT accounting_entry_lines_status_chk
        CHECK (status IN ('active', 'voided', 'archived')),
    CONSTRAINT accounting_entry_lines_amounts_chk
        CHECK (debit >= 0 AND credit >= 0 AND (debit > 0 OR credit > 0))
);

CREATE INDEX IF NOT EXISTS accounting_entry_lines_entry_idx
    ON accounting_entry_lines (entry_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS accounting_entry_lines_account_idx
    ON accounting_entry_lines (account_id)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- payment_gateway_configs
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_gateway_configs (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway             TEXT         NOT NULL,
    merchant_id         TEXT         NULL,
    secrets_kms_ref     TEXT         NULL,
    enabled             BOOLEAN      NOT NULL DEFAULT false,
    config              JSONB        NOT NULL DEFAULT '{}'::JSONB,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT payment_gateway_configs_status_chk
        CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS payment_gateway_configs_gateway_unique
    ON payment_gateway_configs (gateway)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- payment_webhook_idempotency (dedup webhooks)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_webhook_idempotency (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway             TEXT         NOT NULL,
    idempotency_key     TEXT         NOT NULL,
    payload_hash        TEXT         NULL,
    received_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    processed_at        TIMESTAMPTZ  NULL,
    payment_id          UUID         NULL REFERENCES payments(id) ON DELETE SET NULL,
    last_error          TEXT         NULL,
    CONSTRAINT payment_webhook_idempotency_unique
        UNIQUE (gateway, idempotency_key)
);

CREATE INDEX IF NOT EXISTS payment_webhook_idempotency_received_idx
    ON payment_webhook_idempotency (received_at DESC);

-- ----------------------------------------------------------------------------
-- late_fee_runs (idempotente por periodo)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS late_fee_runs (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    period_year     INTEGER      NOT NULL,
    period_month    INTEGER      NOT NULL,
    rate_applied    NUMERIC(7,6) NOT NULL,
    executed_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    executed_by     UUID         NULL REFERENCES users(id),
    charges_created INTEGER      NOT NULL DEFAULT 0,
    status          TEXT         NOT NULL DEFAULT 'completed',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT late_fee_runs_status_chk
        CHECK (status IN ('draft', 'completed', 'cancelled', 'archived')),
    CONSTRAINT late_fee_runs_period_month_chk
        CHECK (period_month >= 1 AND period_month <= 12)
);

CREATE UNIQUE INDEX IF NOT EXISTS late_fee_runs_period_unique
    ON late_fee_runs (period_year, period_month)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- period_closures
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS period_closures (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    period_year     INTEGER      NOT NULL,
    period_month    INTEGER      NOT NULL,
    closed_soft_at  TIMESTAMPTZ  NULL,
    closed_hard_at  TIMESTAMPTZ  NULL,
    closed_by       UUID         NULL REFERENCES users(id),
    notes           TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'open',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ  NULL,
    created_by      UUID         NULL REFERENCES users(id),
    updated_by      UUID         NULL REFERENCES users(id),
    deleted_by      UUID         NULL REFERENCES users(id),
    version         INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT period_closures_status_chk
        CHECK (status IN ('open', 'closed_soft', 'closed_hard', 'archived')),
    CONSTRAINT period_closures_period_month_chk
        CHECK (period_month >= 1 AND period_month <= 12)
);

CREATE UNIQUE INDEX IF NOT EXISTS period_closures_period_unique
    ON period_closures (period_year, period_month)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- paid_in_full_certificates
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS paid_in_full_certificates (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id             UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    issued_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    valid_until         TIMESTAMPTZ  NOT NULL,
    pdf_url             TEXT         NULL,
    pdf_hash            TEXT         NULL,
    signed_by_user_id   UUID         NULL REFERENCES users(id),
    notes               TEXT         NULL,
    status              TEXT         NOT NULL DEFAULT 'issued',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    created_by          UUID         NULL REFERENCES users(id),
    updated_by          UUID         NULL REFERENCES users(id),
    deleted_by          UUID         NULL REFERENCES users(id),
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT paid_in_full_certificates_status_chk
        CHECK (status IN ('issued', 'revoked', 'archived'))
);

CREATE INDEX IF NOT EXISTS paid_in_full_certificates_unit_idx
    ON paid_in_full_certificates (unit_id, issued_at DESC)
    WHERE deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- finance_outbox_events
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finance_outbox_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id      UUID         NOT NULL,
    event_type        TEXT         NOT NULL,
    payload           JSONB        NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts          INTEGER      NOT NULL DEFAULT 0,
    delivered_at      TIMESTAMPTZ  NULL,
    last_error        TEXT         NULL
);

CREATE INDEX IF NOT EXISTS finance_outbox_events_pending_idx
    ON finance_outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;
