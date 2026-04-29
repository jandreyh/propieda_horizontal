-- Queries del modulo finance (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- chart_of_accounts
-- ----------------------------------------------------------------------------

-- name: CreateChartOfAccount :one
-- Crea una cuenta contable nueva en estado 'active'.
INSERT INTO chart_of_accounts (
    code, name, account_type, parent_id,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, 'active', $5, $5
)
RETURNING id, code, name, account_type, parent_id, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetChartOfAccountByID :one
-- Devuelve una cuenta contable por id (no soft-deleted).
SELECT id, code, name, account_type, parent_id, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM chart_of_accounts
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListChartOfAccounts :many
-- Lista cuentas contables activas ordenadas por codigo.
SELECT id, code, name, account_type, parent_id, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM chart_of_accounts
 WHERE deleted_at IS NULL
 ORDER BY code ASC;

-- ----------------------------------------------------------------------------
-- cost_centers
-- ----------------------------------------------------------------------------

-- name: CreateCostCenter :one
-- Crea un centro de costo nuevo en estado 'active'.
INSERT INTO cost_centers (
    code, name, status, created_by, updated_by
) VALUES (
    $1, $2, 'active', $3, $3
)
RETURNING id, code, name, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetCostCenterByID :one
-- Devuelve un centro de costo por id (no soft-deleted).
SELECT id, code, name, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM cost_centers
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListCostCenters :many
-- Lista centros de costo activos ordenados por codigo.
SELECT id, code, name, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM cost_centers
 WHERE deleted_at IS NULL
 ORDER BY code ASC;

-- ----------------------------------------------------------------------------
-- billing_accounts
-- ----------------------------------------------------------------------------

-- name: GetBillingAccountByID :one
-- Devuelve una cuenta de facturacion por id (no soft-deleted).
SELECT id, unit_id, holder_user_id, opened_at, closed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM billing_accounts
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListBillingAccountsByUnitID :many
-- Lista cuentas de facturacion de una unidad.
SELECT id, unit_id, holder_user_id, opened_at, closed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM billing_accounts
 WHERE unit_id = $1
   AND deleted_at IS NULL
 ORDER BY opened_at DESC;

-- ----------------------------------------------------------------------------
-- charges
-- ----------------------------------------------------------------------------

-- name: CreateCharge :one
-- Crea un cargo nuevo en estado 'open' con balance = amount.
INSERT INTO charges (
    billing_account_id, concept, period_year, period_month,
    amount, balance, due_date, cost_center_id, account_id,
    idempotency_key, description, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $5, $6, $7, $8, $9, $10,
    'open', $11, $11
)
RETURNING id, billing_account_id, concept, period_year, period_month,
          amount, balance, due_date, cost_center_id, account_id,
          idempotency_key, description, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetChargeByID :one
-- Devuelve un cargo por id (no soft-deleted).
SELECT id, billing_account_id, concept, period_year, period_month,
       amount, balance, due_date, cost_center_id, account_id,
       idempotency_key, description, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM charges
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListChargesByBillingAccountID :many
-- Lista cargos de una cuenta de facturacion.
SELECT id, billing_account_id, concept, period_year, period_month,
       amount, balance, due_date, cost_center_id, account_id,
       idempotency_key, description, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM charges
 WHERE billing_account_id = $1
   AND deleted_at IS NULL
 ORDER BY due_date ASC;

-- name: UpdateChargeBalance :one
-- Actualiza saldo y status de un cargo con concurrencia optimista.
-- Si newBalance == 0 cambia a 'paid'; si < amount a 'partial'.
UPDATE charges
   SET balance    = sqlc.arg('new_balance'),
       status     = CASE
                      WHEN sqlc.arg('new_balance')::NUMERIC(14,2) <= 0 THEN 'paid'
                      WHEN sqlc.arg('new_balance')::NUMERIC(14,2) < amount THEN 'partial'
                      ELSE status
                    END,
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, billing_account_id, concept, period_year, period_month,
          amount, balance, due_date, cost_center_id, account_id,
          idempotency_key, description, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- payments
-- ----------------------------------------------------------------------------

-- name: CreatePayment :one
-- Crea un pago nuevo. unallocated_amount = amount.
INSERT INTO payments (
    billing_account_id, payer_user_id, method_code,
    gateway, gateway_txn_id, idempotency_key,
    amount, currency, unallocated_amount, captured_at,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $7, now(), $9, $10, $10
)
RETURNING id, billing_account_id, payer_user_id, method_code,
          gateway, gateway_txn_id, idempotency_key,
          amount, currency, unallocated_amount, captured_at,
          settled_at, failure_reason, receipt_number, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPaymentByID :one
-- Devuelve un pago por id (no soft-deleted).
SELECT id, billing_account_id, payer_user_id, method_code,
       gateway, gateway_txn_id, idempotency_key,
       amount, currency, unallocated_amount, captured_at,
       settled_at, failure_reason, receipt_number, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM payments
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetPaymentByIdempotencyKey :one
-- Devuelve un pago por clave de idempotencia.
SELECT id, billing_account_id, payer_user_id, method_code,
       gateway, gateway_txn_id, idempotency_key,
       amount, currency, unallocated_amount, captured_at,
       settled_at, failure_reason, receipt_number, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM payments
 WHERE idempotency_key = $1
   AND deleted_at IS NULL;

-- name: ListPaymentsByBillingAccountID :many
-- Lista pagos de una cuenta de facturacion.
SELECT id, billing_account_id, payer_user_id, method_code,
       gateway, gateway_txn_id, idempotency_key,
       amount, currency, unallocated_amount, captured_at,
       settled_at, failure_reason, receipt_number, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM payments
 WHERE billing_account_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at DESC;

-- name: UpdatePaymentUnallocated :one
-- Actualiza unallocated_amount con concurrencia optimista.
UPDATE payments
   SET unallocated_amount = sqlc.arg('new_unallocated'),
       updated_at         = now(),
       updated_by         = sqlc.arg('updated_by'),
       version            = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, billing_account_id, payer_user_id, method_code,
          gateway, gateway_txn_id, idempotency_key,
          amount, currency, unallocated_amount, captured_at,
          settled_at, failure_reason, receipt_number, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: UpdatePaymentStatus :one
-- Actualiza el status de un pago con concurrencia optimista.
UPDATE payments
   SET status     = sqlc.arg('new_status'),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, billing_account_id, payer_user_id, method_code,
          gateway, gateway_txn_id, idempotency_key,
          amount, currency, unallocated_amount, captured_at,
          settled_at, failure_reason, receipt_number, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- payment_allocations
-- ----------------------------------------------------------------------------

-- name: CreatePaymentAllocation :one
-- Crea una asignacion de pago a cargo.
INSERT INTO payment_allocations (
    payment_id, charge_id, amount,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, 'active', $4, $4
)
RETURNING id, payment_id, charge_id, amount, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListAllocationsByPaymentID :many
-- Lista asignaciones de un pago.
SELECT id, payment_id, charge_id, amount, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM payment_allocations
 WHERE payment_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at ASC;

-- name: ListAllocationsByChargeID :many
-- Lista asignaciones de un cargo.
SELECT id, payment_id, charge_id, amount, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM payment_allocations
 WHERE charge_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at ASC;

-- ----------------------------------------------------------------------------
-- payment_reversals
-- ----------------------------------------------------------------------------

-- name: CreatePaymentReversal :one
-- Crea un reverso de pago en estado 'pending'.
INSERT INTO payment_reversals (
    payment_id, reason, requested_by,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, 'pending', $3, $3
)
RETURNING id, payment_id, reason, requested_by, requested_at,
          approved_by, approved_at, completed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPaymentReversalByID :one
-- Devuelve un reverso por id (no soft-deleted).
SELECT id, payment_id, reason, requested_by, requested_at,
       approved_by, approved_at, completed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM payment_reversals
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ApprovePaymentReversal :one
-- Aprueba un reverso de pago con concurrencia optimista.
UPDATE payment_reversals
   SET status      = 'approved',
       approved_by = sqlc.arg('approved_by'),
       approved_at = now(),
       updated_at  = now(),
       updated_by  = sqlc.arg('approved_by'),
       version     = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
   AND status = 'pending'
RETURNING id, payment_id, reason, requested_by, requested_at,
          approved_by, approved_at, completed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- period_closures
-- ----------------------------------------------------------------------------

-- name: CreatePeriodClosure :one
-- Crea un cierre de periodo soft.
INSERT INTO period_closures (
    period_year, period_month, closed_soft_at, closed_by, notes,
    status, created_by, updated_by
) VALUES (
    $1, $2, now(), $3, $4, 'closed_soft', $3, $3
)
RETURNING id, period_year, period_month, closed_soft_at, closed_hard_at,
          closed_by, notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetPeriodClosureByPeriod :one
-- Devuelve un cierre de periodo por anio y mes.
SELECT id, period_year, period_month, closed_soft_at, closed_hard_at,
       closed_by, notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM period_closures
 WHERE period_year = $1
   AND period_month = $2
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- payment_webhook_idempotency
-- ----------------------------------------------------------------------------

-- name: CreateWebhookIdempotency :one
-- Registra un webhook para deduplicacion.
INSERT INTO payment_webhook_idempotency (
    gateway, idempotency_key, payload_hash
) VALUES (
    $1, $2, $3
)
RETURNING id, gateway, idempotency_key, payload_hash,
          received_at, processed_at, payment_id, last_error;

-- name: MarkWebhookProcessed :exec
-- Marca un webhook como procesado.
UPDATE payment_webhook_idempotency
   SET processed_at = now(),
       payment_id   = sqlc.arg('payment_id'),
       last_error   = NULL
 WHERE id = sqlc.arg('id');

-- name: MarkWebhookFailed :exec
-- Registra un error en un webhook.
UPDATE payment_webhook_idempotency
   SET last_error = sqlc.arg('last_error')
 WHERE id = sqlc.arg('id');

-- ----------------------------------------------------------------------------
-- finance_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueueFinanceOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO finance_outbox_events (
    aggregate_id, event_type, payload, next_attempt_at, attempts
) VALUES (
    $1, $2, $3, now(), 0
)
RETURNING id, aggregate_id, event_type, payload, created_at,
          next_attempt_at, attempts, delivered_at, last_error;

-- name: LockPendingFinanceOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, aggregate_id, event_type, payload, created_at,
       next_attempt_at, attempts, delivered_at, last_error
  FROM finance_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkFinanceOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE finance_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkFinanceOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE finance_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
