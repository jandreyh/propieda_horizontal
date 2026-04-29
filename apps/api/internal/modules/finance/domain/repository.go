// Package domain define los puertos del modulo finance.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/finance/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrAccountNotFound se devuelve cuando una cuenta contable no existe.
var ErrAccountNotFound = errors.New("finance: account not found")

// ErrAccountCodeDuplicate se devuelve cuando el codigo de cuenta ya existe.
var ErrAccountCodeDuplicate = errors.New("finance: account code already exists")

// ErrCostCenterNotFound se devuelve cuando un centro de costo no existe.
var ErrCostCenterNotFound = errors.New("finance: cost center not found")

// ErrCostCenterCodeDuplicate se devuelve cuando el codigo de centro de costo
// ya existe.
var ErrCostCenterCodeDuplicate = errors.New("finance: cost center code already exists")

// ErrBillingAccountNotFound se devuelve cuando una cuenta de facturacion
// no existe.
var ErrBillingAccountNotFound = errors.New("finance: billing account not found")

// ErrBillingAccountDuplicate se devuelve cuando ya existe una cuenta activa
// para la misma unidad y titular.
var ErrBillingAccountDuplicate = errors.New("finance: billing account already exists for unit+holder")

// ErrChargeNotFound se devuelve cuando un cargo no existe.
var ErrChargeNotFound = errors.New("finance: charge not found")

// ErrChargeIdempotencyDuplicate se devuelve cuando la clave de idempotencia
// ya existe.
var ErrChargeIdempotencyDuplicate = errors.New("finance: charge idempotency key already exists")

// ErrPaymentNotFound se devuelve cuando un pago no existe.
var ErrPaymentNotFound = errors.New("finance: payment not found")

// ErrPaymentGatewayTxnDuplicate se devuelve cuando el gateway_txn_id ya
// existe (doble pago en pasarela).
var ErrPaymentGatewayTxnDuplicate = errors.New("finance: gateway transaction already exists")

// ErrPaymentIdempotencyDuplicate se devuelve cuando la clave de
// idempotencia del pago ya existe.
var ErrPaymentIdempotencyDuplicate = errors.New("finance: payment idempotency key already exists")

// ErrAllocationNotFound se devuelve cuando una asignacion de pago no existe.
var ErrAllocationNotFound = errors.New("finance: allocation not found")

// ErrReversalNotFound se devuelve cuando un reverso de pago no existe.
var ErrReversalNotFound = errors.New("finance: reversal not found")

// ErrPeriodClosureNotFound se devuelve cuando un cierre de periodo no existe.
var ErrPeriodClosureNotFound = errors.New("finance: period closure not found")

// ErrPeriodClosureDuplicate se devuelve cuando ya existe un cierre para ese
// periodo.
var ErrPeriodClosureDuplicate = errors.New("finance: period closure already exists")

// ErrWebhookDuplicate se devuelve cuando el par (gateway, idempotency_key)
// ya existe en la tabla de deduplicacion de webhooks.
var ErrWebhookDuplicate = errors.New("finance: webhook already processed")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio.
var ErrVersionConflict = errors.New("finance: version conflict")

// ---------------------------------------------------------------------------
// ChartOfAccountsRepository
// ---------------------------------------------------------------------------

// CreateAccountInput agrupa los datos para crear una cuenta contable.
type CreateAccountInput struct {
	Code        string
	Name        string
	AccountType entities.AccountType
	ParentID    *string
	ActorID     *string
}

// ChartOfAccountsRepository es el puerto que persiste cuentas contables.
type ChartOfAccountsRepository interface {
	Create(ctx context.Context, in CreateAccountInput) (entities.ChartOfAccount, error)
	GetByID(ctx context.Context, id string) (entities.ChartOfAccount, error)
	List(ctx context.Context) ([]entities.ChartOfAccount, error)
}

// ---------------------------------------------------------------------------
// CostCenterRepository
// ---------------------------------------------------------------------------

// CreateCostCenterInput agrupa los datos para crear un centro de costo.
type CreateCostCenterInput struct {
	Code    string
	Name    string
	ActorID *string
}

// CostCenterRepository es el puerto que persiste centros de costo.
type CostCenterRepository interface {
	Create(ctx context.Context, in CreateCostCenterInput) (entities.CostCenter, error)
	GetByID(ctx context.Context, id string) (entities.CostCenter, error)
	List(ctx context.Context) ([]entities.CostCenter, error)
}

// ---------------------------------------------------------------------------
// BillingAccountRepository
// ---------------------------------------------------------------------------

// BillingAccountRepository es el puerto que persiste cuentas de facturacion.
type BillingAccountRepository interface {
	GetByID(ctx context.Context, id string) (entities.BillingAccount, error)
	ListByUnitID(ctx context.Context, unitID string) ([]entities.BillingAccount, error)
}

// ---------------------------------------------------------------------------
// ChargeRepository
// ---------------------------------------------------------------------------

// CreateChargeInput agrupa los datos para crear un cargo.
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

// ChargeRepository es el puerto que persiste cargos.
type ChargeRepository interface {
	Create(ctx context.Context, in CreateChargeInput) (entities.Charge, error)
	GetByID(ctx context.Context, id string) (entities.Charge, error)
	ListByBillingAccountID(ctx context.Context, billingAccountID string) ([]entities.Charge, error)
	// UpdateBalance actualiza el saldo de un cargo con concurrencia
	// optimista. Si balance==0 cambia status a 'paid'; si balance < amount
	// cambia a 'partial'.
	UpdateBalance(ctx context.Context, id string, newBalance float64, expectedVersion int32, actorID string) (entities.Charge, error)
}

// ---------------------------------------------------------------------------
// PaymentRepository
// ---------------------------------------------------------------------------

// CreatePaymentInput agrupa los datos para crear un pago.
type CreatePaymentInput struct {
	BillingAccountID string
	PayerUserID      *string
	MethodCode       string
	Gateway          *string
	GatewayTxnID     *string
	IdempotencyKey   *string
	Amount           float64
	Currency         string
	Status           entities.PaymentStatus
	ActorID          *string
}

// PaymentRepository es el puerto que persiste pagos.
type PaymentRepository interface {
	Create(ctx context.Context, in CreatePaymentInput) (entities.Payment, error)
	GetByID(ctx context.Context, id string) (entities.Payment, error)
	GetByIdempotencyKey(ctx context.Context, key string) (entities.Payment, error)
	ListByBillingAccountID(ctx context.Context, billingAccountID string) ([]entities.Payment, error)
	// UpdateUnallocated actualiza unallocated_amount con concurrencia
	// optimista.
	UpdateUnallocated(ctx context.Context, id string, newUnallocated float64, expectedVersion int32, actorID string) (entities.Payment, error)
	// UpdateStatus actualiza el status con concurrencia optimista.
	UpdateStatus(ctx context.Context, id string, newStatus entities.PaymentStatus, expectedVersion int32, actorID string) (entities.Payment, error)
}

// ---------------------------------------------------------------------------
// PaymentAllocationRepository
// ---------------------------------------------------------------------------

// CreateAllocationInput agrupa los datos para crear una asignacion
// de pago.
type CreateAllocationInput struct {
	PaymentID string
	ChargeID  string
	Amount    float64
	ActorID   *string
}

// PaymentAllocationRepository es el puerto que persiste asignaciones
// de pagos a cargos.
type PaymentAllocationRepository interface {
	Create(ctx context.Context, in CreateAllocationInput) (entities.PaymentAllocation, error)
	ListByPaymentID(ctx context.Context, paymentID string) ([]entities.PaymentAllocation, error)
	ListByChargeID(ctx context.Context, chargeID string) ([]entities.PaymentAllocation, error)
}

// ---------------------------------------------------------------------------
// PaymentReversalRepository
// ---------------------------------------------------------------------------

// CreateReversalInput agrupa los datos para crear un reverso de pago.
type CreateReversalInput struct {
	PaymentID   string
	Reason      string
	RequestedBy string
}

// PaymentReversalRepository es el puerto que persiste reversos de pago.
type PaymentReversalRepository interface {
	Create(ctx context.Context, in CreateReversalInput) (entities.PaymentReversal, error)
	GetByID(ctx context.Context, id string) (entities.PaymentReversal, error)
	// Approve marca un reverso como aprobado con concurrencia optimista.
	Approve(ctx context.Context, id string, approvedBy string, expectedVersion int32) (entities.PaymentReversal, error)
}

// ---------------------------------------------------------------------------
// PeriodClosureRepository
// ---------------------------------------------------------------------------

// CreatePeriodClosureInput agrupa los datos para crear un cierre de
// periodo.
type CreatePeriodClosureInput struct {
	PeriodYear  int32
	PeriodMonth int32
	ClosedBy    *string
	Notes       *string
}

// PeriodClosureRepository es el puerto que persiste cierres de periodo.
type PeriodClosureRepository interface {
	// CreateOrGetSoftClosure crea un cierre soft o retorna el existente.
	CreateOrGetSoftClosure(ctx context.Context, in CreatePeriodClosureInput) (entities.PeriodClosure, error)
	GetByPeriod(ctx context.Context, year, month int32) (entities.PeriodClosure, error)
}

// ---------------------------------------------------------------------------
// WebhookIdempotencyRepository
// ---------------------------------------------------------------------------

// CreateWebhookInput agrupa los datos para registrar un webhook.
type CreateWebhookInput struct {
	Gateway        string
	IdempotencyKey string
	PayloadHash    *string
}

// WebhookIdempotencyRepository es el puerto que persiste registros de
// deduplicacion de webhooks.
type WebhookIdempotencyRepository interface {
	// Create inserta un registro de webhook. Si ya existe, devuelve
	// ErrWebhookDuplicate.
	Create(ctx context.Context, in CreateWebhookInput) (entities.WebhookIdempotency, error)
	// MarkProcessed marca un webhook como procesado exitosamente.
	MarkProcessed(ctx context.Context, id string, paymentID string) error
	// MarkFailed registra un error en un webhook.
	MarkFailed(ctx context.Context, id string, lastError string) error
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	AggregateID string
	EventType   entities.OutboxEventType
	Payload     []byte
}

// OutboxRepository es el puerto que persiste eventos del outbox.
type OutboxRepository interface {
	Enqueue(ctx context.Context, in EnqueueOutboxInput) (entities.OutboxEvent, error)
	LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error
}
