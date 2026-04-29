// Package dto contiene los Data Transfer Objects del modulo finance.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Chart of Accounts
// ---------------------------------------------------------------------------

// CreateAccountRequest es el body de POST /chart-of-accounts.
type CreateAccountRequest struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	AccountType string  `json:"account_type"`
	ParentID    *string `json:"parent_id,omitempty"`
}

// AccountResponse es la representacion HTTP de un ChartOfAccount.
type AccountResponse struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	AccountType string  `json:"account_type"`
	ParentID    *string `json:"parent_id,omitempty"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	Version     int32   `json:"version"`
}

// ListAccountsResponse es el sobre del listado de cuentas.
type ListAccountsResponse struct {
	Items []AccountResponse `json:"items"`
	Total int               `json:"total"`
}

// ---------------------------------------------------------------------------
// Cost Centers
// ---------------------------------------------------------------------------

// CreateCostCenterRequest es el body de POST /cost-centers.
type CreateCostCenterRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// CostCenterResponse es la representacion HTTP de un CostCenter.
type CostCenterResponse struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Version   int32  `json:"version"`
}

// ListCostCentersResponse es el sobre del listado de centros de costo.
type ListCostCentersResponse struct {
	Items []CostCenterResponse `json:"items"`
	Total int                  `json:"total"`
}

// ---------------------------------------------------------------------------
// Charges
// ---------------------------------------------------------------------------

// CreateChargeRequest es el body de POST /charges.
type CreateChargeRequest struct {
	BillingAccountID string  `json:"billing_account_id"`
	Concept          string  `json:"concept"`
	PeriodYear       *int32  `json:"period_year,omitempty"`
	PeriodMonth      *int32  `json:"period_month,omitempty"`
	Amount           float64 `json:"amount"`
	DueDate          string  `json:"due_date"`
	CostCenterID     *string `json:"cost_center_id,omitempty"`
	AccountID        *string `json:"account_id,omitempty"`
	IdempotencyKey   *string `json:"idempotency_key,omitempty"`
	Description      *string `json:"description,omitempty"`
}

// ChargeResponse es la representacion HTTP de un Charge.
type ChargeResponse struct {
	ID               string  `json:"id"`
	BillingAccountID string  `json:"billing_account_id"`
	Concept          string  `json:"concept"`
	PeriodYear       *int32  `json:"period_year,omitempty"`
	PeriodMonth      *int32  `json:"period_month,omitempty"`
	Amount           float64 `json:"amount"`
	Balance          float64 `json:"balance"`
	DueDate          string  `json:"due_date"`
	CostCenterID     *string `json:"cost_center_id,omitempty"`
	AccountID        *string `json:"account_id,omitempty"`
	Description      *string `json:"description,omitempty"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	Version          int32   `json:"version"`
}

// ListChargesResponse es el sobre del listado de cargos.
type ListChargesResponse struct {
	Items []ChargeResponse `json:"items"`
	Total int              `json:"total"`
}

// ---------------------------------------------------------------------------
// Payments
// ---------------------------------------------------------------------------

// CreatePaymentRequest es el body de POST /payments (manual).
type CreatePaymentRequest struct {
	BillingAccountID string  `json:"billing_account_id"`
	PayerUserID      *string `json:"payer_user_id,omitempty"`
	MethodCode       string  `json:"method_code"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	IdempotencyKey   *string `json:"idempotency_key,omitempty"`
}

// PaymentResponse es la representacion HTTP de un Payment.
type PaymentResponse struct {
	ID                string  `json:"id"`
	BillingAccountID  string  `json:"billing_account_id"`
	PayerUserID       *string `json:"payer_user_id,omitempty"`
	MethodCode        string  `json:"method_code"`
	Gateway           *string `json:"gateway,omitempty"`
	GatewayTxnID      *string `json:"gateway_txn_id,omitempty"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	UnallocatedAmount float64 `json:"unallocated_amount"`
	CapturedAt        *string `json:"captured_at,omitempty"`
	ReceiptNumber     *string `json:"receipt_number,omitempty"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	Version           int32   `json:"version"`
}

// ListPaymentsResponse es el sobre del listado de pagos.
type ListPaymentsResponse struct {
	Items []PaymentResponse `json:"items"`
	Total int               `json:"total"`
}

// ---------------------------------------------------------------------------
// Payment Allocations
// ---------------------------------------------------------------------------

// AllocatePaymentRequest es el body de POST /payments/{id}/allocate.
type AllocatePaymentRequest struct {
	Allocations []AllocationLineRequest `json:"allocations"`
}

// AllocationLineRequest es una linea de aplicacion de pago.
type AllocationLineRequest struct {
	ChargeID string  `json:"charge_id"`
	Amount   float64 `json:"amount"`
}

// AllocationResponse es la representacion HTTP de una PaymentAllocation.
type AllocationResponse struct {
	ID        string  `json:"id"`
	PaymentID string  `json:"payment_id"`
	ChargeID  string  `json:"charge_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

// AllocatePaymentResponse es la respuesta de POST /payments/{id}/allocate.
type AllocatePaymentResponse struct {
	Payment     PaymentResponse      `json:"payment"`
	Allocations []AllocationResponse `json:"allocations"`
}

// ---------------------------------------------------------------------------
// Payment Reversals
// ---------------------------------------------------------------------------

// RequestReversalRequest es el body de POST /payments/{id}/reverse.
type RequestReversalRequest struct {
	Reason string `json:"reason"`
}

// ReversalResponse es la representacion HTTP de un PaymentReversal.
type ReversalResponse struct {
	ID          string  `json:"id"`
	PaymentID   string  `json:"payment_id"`
	Reason      string  `json:"reason"`
	RequestedBy string  `json:"requested_by"`
	RequestedAt string  `json:"requested_at"`
	ApprovedBy  *string `json:"approved_by,omitempty"`
	ApprovedAt  *string `json:"approved_at,omitempty"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	Version     int32   `json:"version"`
}

// ---------------------------------------------------------------------------
// Webhook
// ---------------------------------------------------------------------------

// WebhookPayload es el body generico que llega de un webhook de pasarela.
type WebhookPayload struct {
	TransactionID  string  `json:"transaction_id"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	Status         string  `json:"status"`
	IdempotencyKey string  `json:"idempotency_key"`
	MerchantRef    string  `json:"merchant_ref"`
}

// ---------------------------------------------------------------------------
// Period Closure
// ---------------------------------------------------------------------------

// PeriodClosureResponse es la representacion HTTP de un PeriodClosure.
type PeriodClosureResponse struct {
	ID           string  `json:"id"`
	PeriodYear   int32   `json:"period_year"`
	PeriodMonth  int32   `json:"period_month"`
	ClosedSoftAt *string `json:"closed_soft_at,omitempty"`
	ClosedHardAt *string `json:"closed_hard_at,omitempty"`
	ClosedBy     *string `json:"closed_by,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	Version      int32   `json:"version"`
}

// ClosePeriodSoftRequest es el body de POST /periods/{year}/{month}/close-soft.
type ClosePeriodSoftRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// ---------------------------------------------------------------------------
// Billing Account Statement
// ---------------------------------------------------------------------------

// StatementResponse es la respuesta de GET /billing-accounts/{id}/statement.
type StatementResponse struct {
	BillingAccountID string            `json:"billing_account_id"`
	Charges          []ChargeResponse  `json:"charges"`
	Payments         []PaymentResponse `json:"payments"`
	TotalCharged     float64           `json:"total_charged"`
	TotalPaid        float64           `json:"total_paid"`
	TotalBalance     float64           `json:"total_balance"`
}

// ---------------------------------------------------------------------------
// Time formatting helpers
// ---------------------------------------------------------------------------

// FormatTime formatea un time.Time como RFC3339 para JSON.
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatTimePtr formatea un *time.Time como RFC3339 string pointer.
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// FormatDate formatea un time.Time como YYYY-MM-DD para JSON.
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}
