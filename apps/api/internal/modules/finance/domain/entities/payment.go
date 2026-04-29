package entities

import "time"

// PaymentStatus enumera los estados validos de un pago.
type PaymentStatus string

// Possible values for PaymentStatus.
const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusAuthorized PaymentStatus = "authorized"
	PaymentStatusCaptured   PaymentStatus = "captured"
	PaymentStatusSettled    PaymentStatus = "settled"
	PaymentStatusReversed   PaymentStatus = "reversed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusArchived   PaymentStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusAuthorized, PaymentStatusCaptured,
		PaymentStatusSettled, PaymentStatusReversed, PaymentStatusFailed,
		PaymentStatusArchived:
		return true
	}
	return false
}

// Payment representa un pago manual o de pasarela contra una cuenta
// de facturacion.
type Payment struct {
	ID                string
	BillingAccountID  string
	PayerUserID       *string
	MethodCode        string
	Gateway           *string
	GatewayTxnID      *string
	IdempotencyKey    *string
	Amount            float64
	Currency          string
	UnallocatedAmount float64
	CapturedAt        *time.Time
	SettledAt         *time.Time
	FailureReason     *string
	ReceiptNumber     *string
	Status            PaymentStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}

// HasUnallocated indica si el pago tiene monto sin aplicar.
func (p Payment) HasUnallocated() bool {
	return p.UnallocatedAmount > 0
}

// PaymentAllocation representa la aplicacion de un monto de pago a un
// cargo especifico.
type PaymentAllocation struct {
	ID        string
	PaymentID string
	ChargeID  string
	Amount    float64
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int32
}

// ReversalStatus enumera los estados validos de un reverso de pago.
type ReversalStatus string

// Possible values for ReversalStatus.
const (
	ReversalStatusPending   ReversalStatus = "pending"
	ReversalStatusApproved  ReversalStatus = "approved"
	ReversalStatusRejected  ReversalStatus = "rejected"
	ReversalStatusCompleted ReversalStatus = "completed"
	ReversalStatusArchived  ReversalStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s ReversalStatus) IsValid() bool {
	switch s {
	case ReversalStatusPending, ReversalStatusApproved, ReversalStatusRejected,
		ReversalStatusCompleted, ReversalStatusArchived:
		return true
	}
	return false
}

// PaymentReversal representa una solicitud de reverso de pago con
// doble validacion (solicitante + aprobador).
type PaymentReversal struct {
	ID          string
	PaymentID   string
	Reason      string
	RequestedBy string
	RequestedAt time.Time
	ApprovedBy  *string
	ApprovedAt  *time.Time
	CompletedAt *time.Time
	Status      ReversalStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}
