// Package persistence implementa los puertos del modulo finance usando
// el codigo generado por sqlc.
//
// Reglas:
//   - El pool del Tenant DB se obtiene del contexto via tenantctx.FromCtx.
//   - NO se usa database/sql ni SQL inline.
//   - Las usecases que requieren atomicidad multi-tabla pasan un pgx.Tx
//     en el contexto via WithTx(ctx, tx). Si esta presente, los repos lo
//     usan; si no, usan el pool del tenant.
package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/finance/domain"
	"github.com/saas-ph/api/internal/modules/finance/domain/entities"
	financedb "github.com/saas-ph/api/internal/modules/finance/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// --- ctx helper para transaccion ---

type txCtxKey struct{}

// WithTx inyecta una transaccion pgx en el contexto.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

func txFromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	return tx, ok
}

func querier(ctx context.Context) (*financedb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return financedb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("finance: tenant pool is nil")
	}
	return financedb.New(t.Pool), nil
}

// --- ChartOfAccountsRepository ---

// ChartOfAccountsRepository implementa domain.ChartOfAccountsRepository.
type ChartOfAccountsRepository struct{}

// NewChartOfAccountsRepository construye una instancia stateless.
func NewChartOfAccountsRepository() *ChartOfAccountsRepository {
	return &ChartOfAccountsRepository{}
}

// Create implements the repository interface.
func (r *ChartOfAccountsRepository) Create(ctx context.Context, in domain.CreateAccountInput) (entities.ChartOfAccount, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ChartOfAccount{}, err
	}
	row, err := q.CreateChartOfAccount(ctx, financedb.CreateChartOfAccountParams{
		Code:        in.Code,
		Name:        in.Name,
		AccountType: string(in.AccountType),
		ParentID:    uuidToPgtypePtr(in.ParentID),
		CreatedBy:   uuidToPgtypePtr(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.ChartOfAccount{}, domain.ErrAccountCodeDuplicate
		}
		return entities.ChartOfAccount{}, err
	}
	return mapAccount(row), nil
}

// GetByID implements the repository interface.
func (r *ChartOfAccountsRepository) GetByID(ctx context.Context, id string) (entities.ChartOfAccount, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ChartOfAccount{}, err
	}
	row, err := q.GetChartOfAccountByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.ChartOfAccount{}, domain.ErrAccountNotFound
		}
		return entities.ChartOfAccount{}, err
	}
	return mapAccount(row), nil
}

// List implements the repository interface.
func (r *ChartOfAccountsRepository) List(ctx context.Context) ([]entities.ChartOfAccount, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListChartOfAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.ChartOfAccount, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAccount(row))
	}
	return out, nil
}

// --- CostCenterRepository ---

// CostCenterRepository implementa domain.CostCenterRepository.
type CostCenterRepository struct{}

// NewCostCenterRepository construye una instancia stateless.
func NewCostCenterRepository() *CostCenterRepository { return &CostCenterRepository{} }

// Create implements the repository interface.
func (r *CostCenterRepository) Create(ctx context.Context, in domain.CreateCostCenterInput) (entities.CostCenter, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.CostCenter{}, err
	}
	row, err := q.CreateCostCenter(ctx, financedb.CreateCostCenterParams{
		Code:      in.Code,
		Name:      in.Name,
		CreatedBy: uuidToPgtypePtr(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.CostCenter{}, domain.ErrCostCenterCodeDuplicate
		}
		return entities.CostCenter{}, err
	}
	return mapCostCenter(row), nil
}

// GetByID implements the repository interface.
func (r *CostCenterRepository) GetByID(ctx context.Context, id string) (entities.CostCenter, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.CostCenter{}, err
	}
	row, err := q.GetCostCenterByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.CostCenter{}, domain.ErrCostCenterNotFound
		}
		return entities.CostCenter{}, err
	}
	return mapCostCenter(row), nil
}

// List implements the repository interface.
func (r *CostCenterRepository) List(ctx context.Context) ([]entities.CostCenter, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListCostCenters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.CostCenter, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCostCenter(row))
	}
	return out, nil
}

// --- BillingAccountRepository ---

// BillingAccountRepository implementa domain.BillingAccountRepository.
type BillingAccountRepository struct{}

// NewBillingAccountRepository construye una instancia stateless.
func NewBillingAccountRepository() *BillingAccountRepository {
	return &BillingAccountRepository{}
}

// GetByID implements the repository interface.
func (r *BillingAccountRepository) GetByID(ctx context.Context, id string) (entities.BillingAccount, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.BillingAccount{}, err
	}
	row, err := q.GetBillingAccountByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.BillingAccount{}, domain.ErrBillingAccountNotFound
		}
		return entities.BillingAccount{}, err
	}
	return mapBillingAccount(row), nil
}

// ListByUnitID implements the repository interface.
func (r *BillingAccountRepository) ListByUnitID(ctx context.Context, unitID string) ([]entities.BillingAccount, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListBillingAccountsByUnitID(ctx, uuidToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.BillingAccount, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapBillingAccount(row))
	}
	return out, nil
}

// --- ChargeRepository ---

// ChargeRepository implementa domain.ChargeRepository.
type ChargeRepository struct{}

// NewChargeRepository construye una instancia stateless.
func NewChargeRepository() *ChargeRepository { return &ChargeRepository{} }

// Create implements the repository interface.
func (r *ChargeRepository) Create(ctx context.Context, in domain.CreateChargeInput) (entities.Charge, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Charge{}, err
	}
	row, err := q.CreateCharge(ctx, financedb.CreateChargeParams{
		BillingAccountID: uuidToPgtype(in.BillingAccountID),
		Concept:          string(in.Concept),
		PeriodYear:       in.PeriodYear,
		PeriodMonth:      in.PeriodMonth,
		Amount:           float64ToNumeric(in.Amount),
		DueDate:          timeToPgDate(in.DueDate),
		CostCenterID:     uuidToPgtypePtr(in.CostCenterID),
		AccountID:        uuidToPgtypePtr(in.AccountID),
		IdempotencyKey:   in.IdempotencyKey,
		Description:      in.Description,
		CreatedBy:        uuidToPgtypePtr(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Charge{}, domain.ErrChargeIdempotencyDuplicate
		}
		return entities.Charge{}, err
	}
	return mapCharge(row), nil
}

// GetByID implements the repository interface.
func (r *ChargeRepository) GetByID(ctx context.Context, id string) (entities.Charge, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Charge{}, err
	}
	row, err := q.GetChargeByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Charge{}, domain.ErrChargeNotFound
		}
		return entities.Charge{}, err
	}
	return mapCharge(row), nil
}

// ListByBillingAccountID implements the repository interface.
func (r *ChargeRepository) ListByBillingAccountID(ctx context.Context, billingAccountID string) ([]entities.Charge, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListChargesByBillingAccountID(ctx, uuidToPgtype(billingAccountID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Charge, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCharge(row))
	}
	return out, nil
}

// UpdateBalance implements the repository interface.
func (r *ChargeRepository) UpdateBalance(ctx context.Context, id string, newBalance float64, expectedVersion int32, actorID string) (entities.Charge, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Charge{}, err
	}
	row, err := q.UpdateChargeBalance(ctx, financedb.UpdateChargeBalanceParams{
		NewBalance:      float64ToNumeric(newBalance),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Charge{}, domain.ErrVersionConflict
		}
		return entities.Charge{}, err
	}
	return mapCharge(row), nil
}

// --- PaymentRepository ---

// PaymentRepository implementa domain.PaymentRepository.
type PaymentRepository struct{}

// NewPaymentRepository construye una instancia stateless.
func NewPaymentRepository() *PaymentRepository { return &PaymentRepository{} }

// Create implements the repository interface.
func (r *PaymentRepository) Create(ctx context.Context, in domain.CreatePaymentInput) (entities.Payment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Payment{}, err
	}
	row, err := q.CreatePayment(ctx, financedb.CreatePaymentParams{
		BillingAccountID: uuidToPgtype(in.BillingAccountID),
		PayerUserID:      uuidToPgtypePtr(in.PayerUserID),
		MethodCode:       in.MethodCode,
		Gateway:          in.Gateway,
		GatewayTxnID:     in.GatewayTxnID,
		IdempotencyKey:   in.IdempotencyKey,
		Amount:           float64ToNumeric(in.Amount),
		Currency:         in.Currency,
		Status:           string(in.Status),
		CreatedBy:        uuidToPgtypePtr(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Payment{}, domain.ErrPaymentIdempotencyDuplicate
		}
		return entities.Payment{}, err
	}
	return mapPayment(row), nil
}

// GetByID implements the repository interface.
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (entities.Payment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Payment{}, err
	}
	row, err := q.GetPaymentByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Payment{}, domain.ErrPaymentNotFound
		}
		return entities.Payment{}, err
	}
	return mapPayment(row), nil
}

// GetByIdempotencyKey implements the repository interface.
func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (entities.Payment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Payment{}, err
	}
	row, err := q.GetPaymentByIdempotencyKey(ctx, &key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Payment{}, domain.ErrPaymentNotFound
		}
		return entities.Payment{}, err
	}
	return mapPayment(row), nil
}

// ListByBillingAccountID implements the repository interface.
func (r *PaymentRepository) ListByBillingAccountID(ctx context.Context, billingAccountID string) ([]entities.Payment, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPaymentsByBillingAccountID(ctx, uuidToPgtype(billingAccountID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Payment, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPayment(row))
	}
	return out, nil
}

// UpdateUnallocated implements the repository interface.
func (r *PaymentRepository) UpdateUnallocated(ctx context.Context, id string, newUnallocated float64, expectedVersion int32, actorID string) (entities.Payment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Payment{}, err
	}
	row, err := q.UpdatePaymentUnallocated(ctx, financedb.UpdatePaymentUnallocatedParams{
		NewUnallocated:  float64ToNumeric(newUnallocated),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Payment{}, domain.ErrVersionConflict
		}
		return entities.Payment{}, err
	}
	return mapPayment(row), nil
}

// UpdateStatus implements the repository interface.
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id string, newStatus entities.PaymentStatus, expectedVersion int32, actorID string) (entities.Payment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Payment{}, err
	}
	row, err := q.UpdatePaymentStatus(ctx, financedb.UpdatePaymentStatusParams{
		NewStatus:       string(newStatus),
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Payment{}, domain.ErrVersionConflict
		}
		return entities.Payment{}, err
	}
	return mapPayment(row), nil
}

// --- PaymentAllocationRepository ---

// PaymentAllocationRepository implementa domain.PaymentAllocationRepository.
type PaymentAllocationRepository struct{}

// NewPaymentAllocationRepository construye una instancia stateless.
func NewPaymentAllocationRepository() *PaymentAllocationRepository {
	return &PaymentAllocationRepository{}
}

// Create implements the repository interface.
func (r *PaymentAllocationRepository) Create(ctx context.Context, in domain.CreateAllocationInput) (entities.PaymentAllocation, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PaymentAllocation{}, err
	}
	row, err := q.CreatePaymentAllocation(ctx, financedb.CreatePaymentAllocationParams{
		PaymentID: uuidToPgtype(in.PaymentID),
		ChargeID:  uuidToPgtype(in.ChargeID),
		Amount:    float64ToNumeric(in.Amount),
		CreatedBy: uuidToPgtypePtr(in.ActorID),
	})
	if err != nil {
		return entities.PaymentAllocation{}, err
	}
	return mapAllocation(row), nil
}

// ListByPaymentID implements the repository interface.
func (r *PaymentAllocationRepository) ListByPaymentID(ctx context.Context, paymentID string) ([]entities.PaymentAllocation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAllocationsByPaymentID(ctx, uuidToPgtype(paymentID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.PaymentAllocation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAllocation(row))
	}
	return out, nil
}

// ListByChargeID implements the repository interface.
func (r *PaymentAllocationRepository) ListByChargeID(ctx context.Context, chargeID string) ([]entities.PaymentAllocation, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAllocationsByChargeID(ctx, uuidToPgtype(chargeID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.PaymentAllocation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAllocation(row))
	}
	return out, nil
}

// --- PaymentReversalRepository ---

// PaymentReversalRepository implementa domain.PaymentReversalRepository.
type PaymentReversalRepository struct{}

// NewPaymentReversalRepository construye una instancia stateless.
func NewPaymentReversalRepository() *PaymentReversalRepository {
	return &PaymentReversalRepository{}
}

// Create implements the repository interface.
func (r *PaymentReversalRepository) Create(ctx context.Context, in domain.CreateReversalInput) (entities.PaymentReversal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PaymentReversal{}, err
	}
	row, err := q.CreatePaymentReversal(ctx, financedb.CreatePaymentReversalParams{
		PaymentID:   uuidToPgtype(in.PaymentID),
		Reason:      in.Reason,
		RequestedBy: uuidToPgtype(in.RequestedBy),
	})
	if err != nil {
		return entities.PaymentReversal{}, err
	}
	return mapReversal(row), nil
}

// GetByID implements the repository interface.
func (r *PaymentReversalRepository) GetByID(ctx context.Context, id string) (entities.PaymentReversal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PaymentReversal{}, err
	}
	row, err := q.GetPaymentReversalByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PaymentReversal{}, domain.ErrReversalNotFound
		}
		return entities.PaymentReversal{}, err
	}
	return mapReversal(row), nil
}

// Approve implements the repository interface.
func (r *PaymentReversalRepository) Approve(ctx context.Context, id string, approvedBy string, expectedVersion int32) (entities.PaymentReversal, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PaymentReversal{}, err
	}
	row, err := q.ApprovePaymentReversal(ctx, financedb.ApprovePaymentReversalParams{
		ApprovedBy:      uuidToPgtype(approvedBy),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PaymentReversal{}, domain.ErrVersionConflict
		}
		return entities.PaymentReversal{}, err
	}
	return mapReversal(row), nil
}

// --- PeriodClosureRepository ---

// PeriodClosureRepository implementa domain.PeriodClosureRepository.
type PeriodClosureRepository struct{}

// NewPeriodClosureRepository construye una instancia stateless.
func NewPeriodClosureRepository() *PeriodClosureRepository {
	return &PeriodClosureRepository{}
}

// CreateOrGetSoftClosure implements the repository interface.
func (r *PeriodClosureRepository) CreateOrGetSoftClosure(ctx context.Context, in domain.CreatePeriodClosureInput) (entities.PeriodClosure, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PeriodClosure{}, err
	}
	// Try to get existing first.
	existing, err := q.GetPeriodClosureByPeriod(ctx, in.PeriodYear, in.PeriodMonth)
	if err == nil {
		return mapPeriodClosure(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return entities.PeriodClosure{}, err
	}
	// Create new.
	row, err := q.CreatePeriodClosure(ctx, financedb.CreatePeriodClosureParams{
		PeriodYear:  in.PeriodYear,
		PeriodMonth: in.PeriodMonth,
		ClosedBy:    uuidToPgtypePtr(in.ClosedBy),
		Notes:       in.Notes,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.PeriodClosure{}, domain.ErrPeriodClosureDuplicate
		}
		return entities.PeriodClosure{}, err
	}
	return mapPeriodClosure(row), nil
}

// GetByPeriod implements the repository interface.
func (r *PeriodClosureRepository) GetByPeriod(ctx context.Context, year, month int32) (entities.PeriodClosure, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PeriodClosure{}, err
	}
	row, err := q.GetPeriodClosureByPeriod(ctx, year, month)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PeriodClosure{}, domain.ErrPeriodClosureNotFound
		}
		return entities.PeriodClosure{}, err
	}
	return mapPeriodClosure(row), nil
}

// --- WebhookIdempotencyRepository ---

// WebhookIdempotencyRepository implementa domain.WebhookIdempotencyRepository.
type WebhookIdempotencyRepository struct{}

// NewWebhookIdempotencyRepository construye una instancia stateless.
func NewWebhookIdempotencyRepository() *WebhookIdempotencyRepository {
	return &WebhookIdempotencyRepository{}
}

// Create implements the repository interface.
func (r *WebhookIdempotencyRepository) Create(ctx context.Context, in domain.CreateWebhookInput) (entities.WebhookIdempotency, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.WebhookIdempotency{}, err
	}
	row, err := q.CreateWebhookIdempotency(ctx, financedb.CreateWebhookIdempotencyParams{
		Gateway:        in.Gateway,
		IdempotencyKey: in.IdempotencyKey,
		PayloadHash:    in.PayloadHash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.WebhookIdempotency{}, domain.ErrWebhookDuplicate
		}
		return entities.WebhookIdempotency{}, err
	}
	return mapWebhook(row), nil
}

// MarkProcessed implements the repository interface.
func (r *WebhookIdempotencyRepository) MarkProcessed(ctx context.Context, id string, paymentID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	_, err = q.MarkWebhookProcessed(ctx, financedb.MarkWebhookProcessedParams{
		PaymentID: uuidToPgtype(paymentID),
		ID:        uuidToPgtype(id),
	})
	return err
}

// MarkFailed implements the repository interface.
func (r *WebhookIdempotencyRepository) MarkFailed(ctx context.Context, id string, lastError string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	le := lastError
	_, err = q.MarkWebhookFailed(ctx, financedb.MarkWebhookFailedParams{
		LastError: &le,
		ID:        uuidToPgtype(id),
	})
	return err
}

// --- OutboxRepository ---

// OutboxRepository implementa domain.OutboxRepository.
type OutboxRepository struct{}

// NewOutboxRepository construye una instancia stateless.
func NewOutboxRepository() *OutboxRepository { return &OutboxRepository{} }

// Enqueue implements the repository interface.
func (r *OutboxRepository) Enqueue(ctx context.Context, in domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	row, err := q.EnqueueFinanceOutboxEvent(ctx, financedb.EnqueueFinanceOutboxEventParams{
		AggregateID: uuidToPgtype(in.AggregateID),
		EventType:   string(in.EventType),
		Payload:     in.Payload,
	})
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	return mapOutbox(row), nil
}

// LockPending implements the repository interface.
func (r *OutboxRepository) LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.LockPendingFinanceOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]entities.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOutbox(row))
	}
	return out, nil
}

// MarkDelivered implements the repository interface.
func (r *OutboxRepository) MarkDelivered(ctx context.Context, id string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	_, err = q.MarkFinanceOutboxEventDelivered(ctx, uuidToPgtype(id))
	return err
}

// MarkFailed implements the repository interface.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	_, err = q.MarkFinanceOutboxEventFailed(ctx, financedb.MarkFinanceOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
	return err
}

// --- helpers de mapeo ---

func mapAccount(r financedb.ChartOfAccount) entities.ChartOfAccount {
	out := entities.ChartOfAccount{
		ID:          uuidString(r.ID),
		Code:        r.Code,
		Name:        r.Name,
		AccountType: entities.AccountType(r.AccountType),
		Status:      entities.AccountStatus(r.Status),
		CreatedAt:   tsToTime(r.CreatedAt),
		UpdatedAt:   tsToTime(r.UpdatedAt),
		Version:     r.Version,
	}
	if s := uuidStringPtr(r.ParentID); s != nil {
		out.ParentID = s
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapCostCenter(r financedb.CostCenter) entities.CostCenter {
	out := entities.CostCenter{
		ID:        uuidString(r.ID),
		Code:      r.Code,
		Name:      r.Name,
		Status:    entities.CostCenterStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapBillingAccount(r financedb.BillingAccount) entities.BillingAccount {
	out := entities.BillingAccount{
		ID:           uuidString(r.ID),
		UnitID:       uuidString(r.UnitID),
		HolderUserID: uuidString(r.HolderUserID),
		OpenedAt:     tsToTime(r.OpenedAt),
		Status:       entities.BillingAccountStatus(r.Status),
		CreatedAt:    tsToTime(r.CreatedAt),
		UpdatedAt:    tsToTime(r.UpdatedAt),
		Version:      r.Version,
	}
	if r.ClosedAt.Valid {
		t := r.ClosedAt.Time
		out.ClosedAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapCharge(r financedb.Charge) entities.Charge {
	out := entities.Charge{
		ID:               uuidString(r.ID),
		BillingAccountID: uuidString(r.BillingAccountID),
		Concept:          entities.ChargeConcept(r.Concept),
		PeriodYear:       r.PeriodYear,
		PeriodMonth:      r.PeriodMonth,
		DueDate:          dateToTime(r.DueDate),
		IdempotencyKey:   r.IdempotencyKey,
		Description:      r.Description,
		Status:           entities.ChargeStatus(r.Status),
		CreatedAt:        tsToTime(r.CreatedAt),
		UpdatedAt:        tsToTime(r.UpdatedAt),
		Version:          r.Version,
	}
	if r.Amount.Valid {
		f, err := numericToFloat64(r.Amount)
		if err == nil {
			out.Amount = f
		}
	}
	if r.Balance.Valid {
		f, err := numericToFloat64(r.Balance)
		if err == nil {
			out.Balance = f
		}
	}
	if s := uuidStringPtr(r.CostCenterID); s != nil {
		out.CostCenterID = s
	}
	if s := uuidStringPtr(r.AccountID); s != nil {
		out.AccountID = s
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapPayment(r financedb.Payment) entities.Payment {
	out := entities.Payment{
		ID:               uuidString(r.ID),
		BillingAccountID: uuidString(r.BillingAccountID),
		MethodCode:       r.MethodCode,
		Gateway:          r.Gateway,
		GatewayTxnID:     r.GatewayTxnID,
		IdempotencyKey:   r.IdempotencyKey,
		Currency:         r.Currency,
		FailureReason:    r.FailureReason,
		ReceiptNumber:    r.ReceiptNumber,
		Status:           entities.PaymentStatus(r.Status),
		CreatedAt:        tsToTime(r.CreatedAt),
		UpdatedAt:        tsToTime(r.UpdatedAt),
		Version:          r.Version,
	}
	if s := uuidStringPtr(r.PayerUserID); s != nil {
		out.PayerUserID = s
	}
	if r.Amount.Valid {
		f, err := numericToFloat64(r.Amount)
		if err == nil {
			out.Amount = f
		}
	}
	if r.UnallocatedAmount.Valid {
		f, err := numericToFloat64(r.UnallocatedAmount)
		if err == nil {
			out.UnallocatedAmount = f
		}
	}
	if r.CapturedAt.Valid {
		t := r.CapturedAt.Time
		out.CapturedAt = &t
	}
	if r.SettledAt.Valid {
		t := r.SettledAt.Time
		out.SettledAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapAllocation(r financedb.PaymentAllocation) entities.PaymentAllocation {
	out := entities.PaymentAllocation{
		ID:        uuidString(r.ID),
		PaymentID: uuidString(r.PaymentID),
		ChargeID:  uuidString(r.ChargeID),
		Status:    r.Status,
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
	}
	if r.Amount.Valid {
		f, err := numericToFloat64(r.Amount)
		if err == nil {
			out.Amount = f
		}
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapReversal(r financedb.PaymentReversal) entities.PaymentReversal {
	out := entities.PaymentReversal{
		ID:          uuidString(r.ID),
		PaymentID:   uuidString(r.PaymentID),
		Reason:      r.Reason,
		RequestedBy: uuidString(r.RequestedBy),
		RequestedAt: tsToTime(r.RequestedAt),
		Status:      entities.ReversalStatus(r.Status),
		CreatedAt:   tsToTime(r.CreatedAt),
		UpdatedAt:   tsToTime(r.UpdatedAt),
		Version:     r.Version,
	}
	if s := uuidStringPtr(r.ApprovedBy); s != nil {
		out.ApprovedBy = s
	}
	if r.ApprovedAt.Valid {
		t := r.ApprovedAt.Time
		out.ApprovedAt = &t
	}
	if r.CompletedAt.Valid {
		t := r.CompletedAt.Time
		out.CompletedAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapPeriodClosure(r financedb.PeriodClosure) entities.PeriodClosure {
	out := entities.PeriodClosure{
		ID:          uuidString(r.ID),
		PeriodYear:  r.PeriodYear,
		PeriodMonth: r.PeriodMonth,
		Notes:       r.Notes,
		Status:      entities.PeriodClosureStatus(r.Status),
		CreatedAt:   tsToTime(r.CreatedAt),
		UpdatedAt:   tsToTime(r.UpdatedAt),
		Version:     r.Version,
	}
	if r.ClosedSoftAt.Valid {
		t := r.ClosedSoftAt.Time
		out.ClosedSoftAt = &t
	}
	if r.ClosedHardAt.Valid {
		t := r.ClosedHardAt.Time
		out.ClosedHardAt = &t
	}
	if s := uuidStringPtr(r.ClosedBy); s != nil {
		out.ClosedBy = s
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func mapWebhook(r financedb.PaymentWebhookIdempotency) entities.WebhookIdempotency {
	out := entities.WebhookIdempotency{
		ID:             uuidString(r.ID),
		Gateway:        r.Gateway,
		IdempotencyKey: r.IdempotencyKey,
		PayloadHash:    r.PayloadHash,
		ReceivedAt:     tsToTime(r.ReceivedAt),
		LastError:      r.LastError,
	}
	if r.ProcessedAt.Valid {
		t := r.ProcessedAt.Time
		out.ProcessedAt = &t
	}
	if s := uuidStringPtr(r.PaymentID); s != nil {
		out.PaymentID = s
	}
	return out
}

func mapOutbox(r financedb.FinanceOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:            uuidString(r.ID),
		AggregateID:   uuidString(r.AggregateID),
		EventType:     entities.OutboxEventType(r.EventType),
		Payload:       r.Payload,
		CreatedAt:     tsToTime(r.CreatedAt),
		NextAttemptAt: tsToTime(r.NextAttemptAt),
		Attempts:      r.Attempts,
		LastError:     r.LastError,
	}
	if r.DeliveredAt.Valid {
		t := r.DeliveredAt.Time
		out.DeliveredAt = &t
	}
	return out
}

// --- pgtype helpers ---

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func dateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

func timeToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	v, err := u.Value()
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	if s == "" {
		return nil
	}
	return &s
}

func uuidToPgtype(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

func uuidToPgtypePtr(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{Valid: false}
	}
	return uuidToPgtype(*s)
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(f); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
}

func numericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, errors.New("numeric is null")
	}
	f64, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f64.Float64, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

// Compile-time checks: each repo implements the domain port.
var (
	_ domain.ChartOfAccountsRepository    = (*ChartOfAccountsRepository)(nil)
	_ domain.CostCenterRepository         = (*CostCenterRepository)(nil)
	_ domain.BillingAccountRepository     = (*BillingAccountRepository)(nil)
	_ domain.ChargeRepository             = (*ChargeRepository)(nil)
	_ domain.PaymentRepository            = (*PaymentRepository)(nil)
	_ domain.PaymentAllocationRepository  = (*PaymentAllocationRepository)(nil)
	_ domain.PaymentReversalRepository    = (*PaymentReversalRepository)(nil)
	_ domain.PeriodClosureRepository      = (*PeriodClosureRepository)(nil)
	_ domain.WebhookIdempotencyRepository = (*WebhookIdempotencyRepository)(nil)
	_ domain.OutboxRepository             = (*OutboxRepository)(nil)
)
