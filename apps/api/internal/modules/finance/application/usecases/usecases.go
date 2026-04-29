// Package usecases orquesta la logica de aplicacion del modulo finance.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/finance/domain"
	"github.com/saas-ph/api/internal/modules/finance/domain/entities"
	"github.com/saas-ph/api/internal/modules/finance/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// CreateAccount
// ---------------------------------------------------------------------------

// CreateAccount crea una cuenta en el plan de cuentas.
type CreateAccount struct {
	Accounts domain.ChartOfAccountsRepository
}

// CreateAccountInput es el input del usecase.
type CreateAccountInput struct {
	Code        string
	Name        string
	AccountType entities.AccountType
	ParentID    *string
	ActorID     *string
}

// Execute valida y delega al repo.
func (u CreateAccount) Execute(ctx context.Context, in CreateAccountInput) (entities.ChartOfAccount, error) {
	if err := policies.ValidateAccountCode(in.Code); err != nil {
		return entities.ChartOfAccount{}, apperrors.BadRequest("code: " + err.Error())
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.ChartOfAccount{}, apperrors.BadRequest("name is required")
	}
	if !in.AccountType.IsValid() {
		return entities.ChartOfAccount{}, apperrors.BadRequest("account_type: invalid account type")
	}
	if in.ParentID != nil {
		if err := policies.ValidateUUID(*in.ParentID); err != nil {
			return entities.ChartOfAccount{}, apperrors.BadRequest("parent_id: " + err.Error())
		}
	}
	acct, err := u.Accounts.Create(ctx, domain.CreateAccountInput{
		Code:        strings.TrimSpace(in.Code),
		Name:        strings.TrimSpace(in.Name),
		AccountType: in.AccountType,
		ParentID:    in.ParentID,
		ActorID:     in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrAccountCodeDuplicate) {
			return entities.ChartOfAccount{}, apperrors.Conflict("account code already exists")
		}
		return entities.ChartOfAccount{}, apperrors.Internal("failed to create account")
	}
	return acct, nil
}

// ---------------------------------------------------------------------------
// ListAccounts
// ---------------------------------------------------------------------------

// ListAccounts lista las cuentas del plan de cuentas.
type ListAccounts struct {
	Accounts domain.ChartOfAccountsRepository
}

// Execute delega al repo.
func (u ListAccounts) Execute(ctx context.Context) ([]entities.ChartOfAccount, error) {
	out, err := u.Accounts.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list accounts")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// CreateCostCenter
// ---------------------------------------------------------------------------

// CreateCostCenter crea un centro de costo.
type CreateCostCenter struct {
	CostCenters domain.CostCenterRepository
}

// CreateCostCenterInput es el input del usecase.
type CreateCostCenterInput struct {
	Code    string
	Name    string
	ActorID *string
}

// Execute valida y delega al repo.
func (u CreateCostCenter) Execute(ctx context.Context, in CreateCostCenterInput) (entities.CostCenter, error) {
	if err := policies.ValidateCostCenterCode(in.Code); err != nil {
		return entities.CostCenter{}, apperrors.BadRequest("code: " + err.Error())
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.CostCenter{}, apperrors.BadRequest("name is required")
	}
	cc, err := u.CostCenters.Create(ctx, domain.CreateCostCenterInput{
		Code:    strings.TrimSpace(in.Code),
		Name:    strings.TrimSpace(in.Name),
		ActorID: in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCostCenterCodeDuplicate) {
			return entities.CostCenter{}, apperrors.Conflict("cost center code already exists")
		}
		return entities.CostCenter{}, apperrors.Internal("failed to create cost center")
	}
	return cc, nil
}

// ---------------------------------------------------------------------------
// ListCostCenters
// ---------------------------------------------------------------------------

// ListCostCenters lista los centros de costo.
type ListCostCenters struct {
	CostCenters domain.CostCenterRepository
}

// Execute delega al repo.
func (u ListCostCenters) Execute(ctx context.Context) ([]entities.CostCenter, error) {
	out, err := u.CostCenters.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list cost centers")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// CreateCharge
// ---------------------------------------------------------------------------

// CreateCharge crea un cargo individual contra una cuenta de facturacion.
type CreateCharge struct {
	Charges         domain.ChargeRepository
	BillingAccounts domain.BillingAccountRepository
	Outbox          domain.OutboxRepository
	TxRunner        TxRunner
}

// CreateChargeInput es el input del usecase.
type CreateChargeInput struct {
	BillingAccountID string
	Concept          entities.ChargeConcept
	PeriodYear       *int32
	PeriodMonth      *int32
	Amount           float64
	DueDate          time.Time
	CostCenterID     *string
	AccountID        *string
	IdempotencyKey   *string
	Description      *string
	ActorID          *string
}

// Execute valida y delega.
func (u CreateCharge) Execute(ctx context.Context, in CreateChargeInput) (entities.Charge, error) {
	if err := policies.ValidateUUID(in.BillingAccountID); err != nil {
		return entities.Charge{}, apperrors.BadRequest("billing_account_id: " + err.Error())
	}
	if !in.Concept.IsValid() {
		return entities.Charge{}, apperrors.BadRequest("concept: invalid charge concept")
	}
	if err := policies.ValidatePositiveAmount(in.Amount); err != nil {
		return entities.Charge{}, apperrors.BadRequest("amount: " + err.Error())
	}
	if in.PeriodYear != nil && in.PeriodMonth != nil {
		if err := policies.ValidatePeriod(*in.PeriodYear, *in.PeriodMonth); err != nil {
			return entities.Charge{}, apperrors.BadRequest(err.Error())
		}
	}

	// Verify billing account exists.
	_, err := u.BillingAccounts.GetByID(ctx, in.BillingAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrBillingAccountNotFound) {
			return entities.Charge{}, apperrors.NotFound("billing account not found")
		}
		return entities.Charge{}, apperrors.Internal("failed to load billing account")
	}

	var created entities.Charge
	run := func(txCtx context.Context) error {
		charge, createErr := u.Charges.Create(txCtx, domain.CreateChargeInput{
			BillingAccountID: in.BillingAccountID,
			Concept:          in.Concept,
			PeriodYear:       in.PeriodYear,
			PeriodMonth:      in.PeriodMonth,
			Amount:           in.Amount,
			DueDate:          in.DueDate,
			CostCenterID:     in.CostCenterID,
			AccountID:        in.AccountID,
			IdempotencyKey:   in.IdempotencyKey,
			Description:      in.Description,
			ActorID:          in.ActorID,
		})
		if createErr != nil {
			return createErr
		}

		payload, _ := json.Marshal(map[string]any{
			"charge_id":          charge.ID,
			"billing_account_id": charge.BillingAccountID,
			"concept":            string(charge.Concept),
			"amount":             charge.Amount,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: charge.ID,
			EventType:   entities.OutboxEventChargeCreated,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		created = charge
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrChargeIdempotencyDuplicate) {
				return entities.Charge{}, apperrors.Conflict("charge idempotency key already exists")
			}
			return entities.Charge{}, apperrors.Internal("failed to create charge")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrChargeIdempotencyDuplicate) {
				return entities.Charge{}, apperrors.Conflict("charge idempotency key already exists")
			}
			return entities.Charge{}, apperrors.Internal("failed to create charge")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// ListCharges
// ---------------------------------------------------------------------------

// ListCharges lista los cargos de una cuenta de facturacion.
type ListCharges struct {
	Charges domain.ChargeRepository
}

// Execute delega al repo.
func (u ListCharges) Execute(ctx context.Context, billingAccountID string) ([]entities.Charge, error) {
	if err := policies.ValidateUUID(billingAccountID); err != nil {
		return nil, apperrors.BadRequest("billing_account_id: " + err.Error())
	}
	out, err := u.Charges.ListByBillingAccountID(ctx, billingAccountID)
	if err != nil {
		return nil, apperrors.Internal("failed to list charges")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// CreatePayment
// ---------------------------------------------------------------------------

// CreatePayment crea un pago manual contra una cuenta de facturacion.
type CreatePayment struct {
	Payments        domain.PaymentRepository
	BillingAccounts domain.BillingAccountRepository
	Outbox          domain.OutboxRepository
	TxRunner        TxRunner
	Now             func() time.Time
}

// CreatePaymentInput es el input del usecase.
type CreatePaymentInput struct {
	BillingAccountID string
	PayerUserID      *string
	MethodCode       string
	Amount           float64
	Currency         string
	IdempotencyKey   *string
	ActorID          *string
}

// Execute valida y delega.
func (u CreatePayment) Execute(ctx context.Context, in CreatePaymentInput) (entities.Payment, error) {
	if err := policies.ValidateUUID(in.BillingAccountID); err != nil {
		return entities.Payment{}, apperrors.BadRequest("billing_account_id: " + err.Error())
	}
	if strings.TrimSpace(in.MethodCode) == "" {
		return entities.Payment{}, apperrors.BadRequest("method_code is required")
	}
	if err := policies.ValidatePositiveAmount(in.Amount); err != nil {
		return entities.Payment{}, apperrors.BadRequest("amount: " + err.Error())
	}
	currency := in.Currency
	if currency == "" {
		currency = "COP"
	}
	if err := policies.ValidateCurrency(currency); err != nil {
		return entities.Payment{}, apperrors.BadRequest("currency: " + err.Error())
	}

	// Verify billing account exists.
	_, err := u.BillingAccounts.GetByID(ctx, in.BillingAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrBillingAccountNotFound) {
			return entities.Payment{}, apperrors.NotFound("billing account not found")
		}
		return entities.Payment{}, apperrors.Internal("failed to load billing account")
	}

	var created entities.Payment
	run := func(txCtx context.Context) error {
		payment, createErr := u.Payments.Create(txCtx, domain.CreatePaymentInput{
			BillingAccountID: in.BillingAccountID,
			PayerUserID:      in.PayerUserID,
			MethodCode:       strings.TrimSpace(in.MethodCode),
			Amount:           in.Amount,
			Currency:         currency,
			Status:           entities.PaymentStatusCaptured,
			IdempotencyKey:   in.IdempotencyKey,
			ActorID:          in.ActorID,
		})
		if createErr != nil {
			return createErr
		}

		payload, _ := json.Marshal(map[string]any{
			"payment_id":         payment.ID,
			"billing_account_id": payment.BillingAccountID,
			"amount":             payment.Amount,
			"method_code":        payment.MethodCode,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: payment.ID,
			EventType:   entities.OutboxEventPaymentCaptured,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		created = payment
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrPaymentIdempotencyDuplicate) {
				return entities.Payment{}, apperrors.Conflict("payment idempotency key already exists")
			}
			return entities.Payment{}, apperrors.Internal("failed to create payment")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrPaymentIdempotencyDuplicate) {
				return entities.Payment{}, apperrors.Conflict("payment idempotency key already exists")
			}
			return entities.Payment{}, apperrors.Internal("failed to create payment")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// ListPayments
// ---------------------------------------------------------------------------

// ListPayments lista los pagos de una cuenta de facturacion.
type ListPayments struct {
	Payments domain.PaymentRepository
}

// Execute delega al repo.
func (u ListPayments) Execute(ctx context.Context, billingAccountID string) ([]entities.Payment, error) {
	if err := policies.ValidateUUID(billingAccountID); err != nil {
		return nil, apperrors.BadRequest("billing_account_id: " + err.Error())
	}
	out, err := u.Payments.ListByBillingAccountID(ctx, billingAccountID)
	if err != nil {
		return nil, apperrors.Internal("failed to list payments")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// AllocatePayment
// ---------------------------------------------------------------------------

// AllocatePayment aplica un pago a uno o mas cargos dentro de una TX.
type AllocatePayment struct {
	Payments    domain.PaymentRepository
	Charges     domain.ChargeRepository
	Allocations domain.PaymentAllocationRepository
	Outbox      domain.OutboxRepository
	TxRunner    TxRunner
}

// AllocationLine es una linea de aplicacion.
type AllocationLine struct {
	ChargeID string
	Amount   float64
}

// AllocatePaymentInput es el input del usecase.
type AllocatePaymentInput struct {
	PaymentID   string
	Allocations []AllocationLine
	ActorID     string
}

// AllocatePaymentResult es el output del usecase.
type AllocatePaymentResult struct {
	Payment     entities.Payment
	Allocations []entities.PaymentAllocation
}

// Execute valida y delega.
func (u AllocatePayment) Execute(ctx context.Context, in AllocatePaymentInput) (AllocatePaymentResult, error) {
	if err := policies.ValidateUUID(in.PaymentID); err != nil {
		return AllocatePaymentResult{}, apperrors.BadRequest("payment_id: " + err.Error())
	}
	if len(in.Allocations) == 0 {
		return AllocatePaymentResult{}, apperrors.BadRequest("allocations must not be empty")
	}

	var totalAllocating float64
	for _, a := range in.Allocations {
		if err := policies.ValidateUUID(a.ChargeID); err != nil {
			return AllocatePaymentResult{}, apperrors.BadRequest("charge_id: " + err.Error())
		}
		if err := policies.ValidatePositiveAmount(a.Amount); err != nil {
			return AllocatePaymentResult{}, apperrors.BadRequest("allocation amount: " + err.Error())
		}
		totalAllocating += a.Amount
	}

	payment, err := u.Payments.GetByID(ctx, in.PaymentID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return AllocatePaymentResult{}, apperrors.NotFound("payment not found")
		}
		return AllocatePaymentResult{}, apperrors.Internal("failed to load payment")
	}

	if err := policies.CanAllocatePayment(payment, totalAllocating); err != nil {
		return AllocatePaymentResult{}, apperrors.BadRequest(err.Error())
	}

	var result AllocatePaymentResult
	run := func(txCtx context.Context) error {
		currentPayment := payment
		allocations := make([]entities.PaymentAllocation, 0, len(in.Allocations))

		for _, line := range in.Allocations {
			charge, cErr := u.Charges.GetByID(txCtx, line.ChargeID)
			if cErr != nil {
				if errors.Is(cErr, domain.ErrChargeNotFound) {
					return apperrors.NotFound("charge not found: " + line.ChargeID)
				}
				return cErr
			}
			if err := policies.CanAllocateCharge(charge, line.Amount); err != nil {
				return apperrors.BadRequest(err.Error())
			}

			alloc, aErr := u.Allocations.Create(txCtx, domain.CreateAllocationInput{
				PaymentID: currentPayment.ID,
				ChargeID:  line.ChargeID,
				Amount:    line.Amount,
				ActorID:   &in.ActorID,
			})
			if aErr != nil {
				return aErr
			}
			allocations = append(allocations, alloc)

			newBalance := charge.Balance - line.Amount
			if _, uErr := u.Charges.UpdateBalance(txCtx, charge.ID, newBalance, charge.Version, in.ActorID); uErr != nil {
				return uErr
			}
		}

		newUnallocated := currentPayment.UnallocatedAmount - totalAllocating
		updatedPayment, uErr := u.Payments.UpdateUnallocated(txCtx, currentPayment.ID, newUnallocated, currentPayment.Version, in.ActorID)
		if uErr != nil {
			return uErr
		}

		payload, _ := json.Marshal(map[string]any{
			"payment_id":      updatedPayment.ID,
			"allocations":     len(allocations),
			"total_allocated": totalAllocating,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: updatedPayment.ID,
			EventType:   entities.OutboxEventPaymentAllocated,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		result = AllocatePaymentResult{Payment: updatedPayment, Allocations: allocations}
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			var p apperrors.Problem
			if errors.As(err, &p) {
				return AllocatePaymentResult{}, err
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return AllocatePaymentResult{}, mapVersionConflict()
			}
			return AllocatePaymentResult{}, apperrors.Internal("failed to allocate payment")
		}
	} else {
		if err := run(ctx); err != nil {
			var p apperrors.Problem
			if errors.As(err, &p) {
				return AllocatePaymentResult{}, err
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return AllocatePaymentResult{}, mapVersionConflict()
			}
			return AllocatePaymentResult{}, apperrors.Internal("failed to allocate payment")
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// ProcessWebhook
// ---------------------------------------------------------------------------

// ProcessWebhook procesa un webhook de pasarela de pago con idempotencia.
type ProcessWebhook struct {
	Webhooks        domain.WebhookIdempotencyRepository
	Payments        domain.PaymentRepository
	BillingAccounts domain.BillingAccountRepository
	Outbox          domain.OutboxRepository
	TxRunner        TxRunner
	Now             func() time.Time
}

// WebhookInput es el input del usecase.
type WebhookInput struct {
	Gateway        string
	TransactionID  string
	Amount         float64
	Currency       string
	Status         string
	IdempotencyKey string
	MerchantRef    string
	PayloadHash    *string
}

// Execute procesa el webhook de forma idempotente.
func (u ProcessWebhook) Execute(ctx context.Context, in WebhookInput) error {
	if strings.TrimSpace(in.Gateway) == "" {
		return apperrors.BadRequest("gateway is required")
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return apperrors.BadRequest("idempotency_key is required")
	}

	// Register the webhook (dedup).
	wh, err := u.Webhooks.Create(ctx, domain.CreateWebhookInput{
		Gateway:        in.Gateway,
		IdempotencyKey: in.IdempotencyKey,
		PayloadHash:    in.PayloadHash,
	})
	if err != nil {
		if errors.Is(err, domain.ErrWebhookDuplicate) {
			// Already processed -- return 200 OK (idempotent).
			return nil
		}
		return apperrors.Internal("failed to register webhook")
	}

	// Process: for now, we just mark the webhook as processed.
	// In a real implementation, we would look up the billing account
	// from MerchantRef, create a payment, etc.
	if err := u.Webhooks.MarkProcessed(ctx, wh.ID, ""); err != nil {
		return apperrors.Internal("failed to mark webhook processed")
	}

	return nil
}

// ---------------------------------------------------------------------------
// RequestReversal
// ---------------------------------------------------------------------------

// RequestReversal solicita un reverso de pago (requiere aprobacion posterior).
type RequestReversal struct {
	Payments  domain.PaymentRepository
	Reversals domain.PaymentReversalRepository
}

// RequestReversalInput es el input del usecase.
type RequestReversalInput struct {
	PaymentID   string
	Reason      string
	RequestedBy string
}

// Execute valida y delega.
func (u RequestReversal) Execute(ctx context.Context, in RequestReversalInput) (entities.PaymentReversal, error) {
	if err := policies.ValidateUUID(in.PaymentID); err != nil {
		return entities.PaymentReversal{}, apperrors.BadRequest("payment_id: " + err.Error())
	}
	if strings.TrimSpace(in.Reason) == "" {
		return entities.PaymentReversal{}, apperrors.BadRequest("reason is required")
	}

	payment, err := u.Payments.GetByID(ctx, in.PaymentID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return entities.PaymentReversal{}, apperrors.NotFound("payment not found")
		}
		return entities.PaymentReversal{}, apperrors.Internal("failed to load payment")
	}
	if err := policies.CanReversePayment(payment); err != nil {
		return entities.PaymentReversal{}, apperrors.BadRequest(err.Error())
	}

	reversal, err := u.Reversals.Create(ctx, domain.CreateReversalInput{
		PaymentID:   in.PaymentID,
		Reason:      strings.TrimSpace(in.Reason),
		RequestedBy: in.RequestedBy,
	})
	if err != nil {
		return entities.PaymentReversal{}, apperrors.Internal("failed to create reversal")
	}
	return reversal, nil
}

// ---------------------------------------------------------------------------
// ApproveReversal
// ---------------------------------------------------------------------------

// ApproveReversal aprueba un reverso de pago pendiente y marca el pago
// como reversed.
type ApproveReversal struct {
	Reversals domain.PaymentReversalRepository
	Payments  domain.PaymentRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
}

// ApproveReversalInput es el input del usecase.
type ApproveReversalInput struct {
	PaymentID  string
	ReversalID string
	ApprovedBy string
}

// Execute valida y delega.
func (u ApproveReversal) Execute(ctx context.Context, in ApproveReversalInput) (entities.PaymentReversal, error) {
	if err := policies.ValidateUUID(in.ReversalID); err != nil {
		return entities.PaymentReversal{}, apperrors.BadRequest("reversal_id: " + err.Error())
	}

	reversal, err := u.Reversals.GetByID(ctx, in.ReversalID)
	if err != nil {
		if errors.Is(err, domain.ErrReversalNotFound) {
			return entities.PaymentReversal{}, apperrors.NotFound("reversal not found")
		}
		return entities.PaymentReversal{}, apperrors.Internal("failed to load reversal")
	}
	if reversal.Status != entities.ReversalStatusPending {
		return entities.PaymentReversal{}, apperrors.Conflict("reversal is not pending")
	}

	var approved entities.PaymentReversal
	run := func(txCtx context.Context) error {
		result, aErr := u.Reversals.Approve(txCtx, reversal.ID, in.ApprovedBy, reversal.Version)
		if aErr != nil {
			return aErr
		}

		// Mark the payment as reversed.
		payment, pErr := u.Payments.GetByID(txCtx, reversal.PaymentID)
		if pErr != nil {
			return pErr
		}
		if _, sErr := u.Payments.UpdateStatus(txCtx, payment.ID, entities.PaymentStatusReversed, payment.Version, in.ApprovedBy); sErr != nil {
			return sErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reversal_id": result.ID,
			"payment_id":  result.PaymentID,
			"approved_by": in.ApprovedBy,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: result.ID,
			EventType:   entities.OutboxEventPaymentReversed,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		approved = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.PaymentReversal{}, mapVersionConflict()
			}
			return entities.PaymentReversal{}, apperrors.Internal("failed to approve reversal")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.PaymentReversal{}, mapVersionConflict()
			}
			return entities.PaymentReversal{}, apperrors.Internal("failed to approve reversal")
		}
	}
	return approved, nil
}

// ---------------------------------------------------------------------------
// ClosePeriodSoft
// ---------------------------------------------------------------------------

// ClosePeriodSoft cierra un periodo contable de forma soft.
type ClosePeriodSoft struct {
	Closures domain.PeriodClosureRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
	Now      func() time.Time
}

// ClosePeriodSoftInput es el input del usecase.
type ClosePeriodSoftInput struct {
	PeriodYear  int32
	PeriodMonth int32
	ActorID     string
	Notes       *string
}

// Execute valida y delega.
func (u ClosePeriodSoft) Execute(ctx context.Context, in ClosePeriodSoftInput) (entities.PeriodClosure, error) {
	if err := policies.ValidatePeriod(in.PeriodYear, in.PeriodMonth); err != nil {
		return entities.PeriodClosure{}, apperrors.BadRequest(err.Error())
	}

	var closed entities.PeriodClosure
	run := func(txCtx context.Context) error {
		closure, err := u.Closures.CreateOrGetSoftClosure(txCtx, domain.CreatePeriodClosureInput{
			PeriodYear:  in.PeriodYear,
			PeriodMonth: in.PeriodMonth,
			ClosedBy:    &in.ActorID,
			Notes:       in.Notes,
		})
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"period_closure_id": closure.ID,
			"period_year":       closure.PeriodYear,
			"period_month":      closure.PeriodMonth,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: closure.ID,
			EventType:   entities.OutboxEventPeriodClosedSoft,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		closed = closure
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrPeriodClosureDuplicate) {
				return entities.PeriodClosure{}, apperrors.Conflict("period already closed")
			}
			return entities.PeriodClosure{}, apperrors.Internal("failed to close period")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrPeriodClosureDuplicate) {
				return entities.PeriodClosure{}, apperrors.Conflict("period already closed")
			}
			return entities.PeriodClosure{}, apperrors.Internal("failed to close period")
		}
	}
	return closed, nil
}

// ---------------------------------------------------------------------------
// GetStatement
// ---------------------------------------------------------------------------

// GetStatement devuelve el estado de cuenta de un billing account.
type GetStatement struct {
	BillingAccounts domain.BillingAccountRepository
	Charges         domain.ChargeRepository
	Payments        domain.PaymentRepository
}

// StatementResult es el output del usecase.
type StatementResult struct {
	BillingAccountID string
	Charges          []entities.Charge
	Payments         []entities.Payment
	TotalCharged     float64
	TotalPaid        float64
	TotalBalance     float64
}

// Execute valida y delega.
func (u GetStatement) Execute(ctx context.Context, billingAccountID string) (StatementResult, error) {
	if err := policies.ValidateUUID(billingAccountID); err != nil {
		return StatementResult{}, apperrors.BadRequest("billing_account_id: " + err.Error())
	}

	_, err := u.BillingAccounts.GetByID(ctx, billingAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrBillingAccountNotFound) {
			return StatementResult{}, apperrors.NotFound("billing account not found")
		}
		return StatementResult{}, apperrors.Internal("failed to load billing account")
	}

	charges, err := u.Charges.ListByBillingAccountID(ctx, billingAccountID)
	if err != nil {
		return StatementResult{}, apperrors.Internal("failed to list charges")
	}

	payments, err := u.Payments.ListByBillingAccountID(ctx, billingAccountID)
	if err != nil {
		return StatementResult{}, apperrors.Internal("failed to list payments")
	}

	var totalCharged, totalPaid, totalBalance float64
	for _, c := range charges {
		totalCharged += c.Amount
		totalBalance += c.Balance
	}
	for _, p := range payments {
		if p.Status == entities.PaymentStatusCaptured ||
			p.Status == entities.PaymentStatusSettled {
			totalPaid += p.Amount
		}
	}

	return StatementResult{
		BillingAccountID: billingAccountID,
		Charges:          charges,
		Payments:         payments,
		TotalCharged:     totalCharged,
		TotalPaid:        totalPaid,
		TotalBalance:     totalBalance,
	}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}
