package entities

import "time"

// PaymentStatus enumera los estados validos de un pago de reserva.
type PaymentStatus string

const (
	// PaymentStatusPending indica que el pago esta pendiente.
	PaymentStatusPending PaymentStatus = "pending"
	// PaymentStatusPaid indica que el pago fue realizado.
	PaymentStatusPaid PaymentStatus = "paid"
	// PaymentStatusRefunded indica que el pago fue reembolsado.
	PaymentStatusRefunded PaymentStatus = "refunded"
	// PaymentStatusForfeited indica que el deposito fue retenido.
	PaymentStatusForfeited PaymentStatus = "forfeited"
	// PaymentStatusArchived indica que el pago fue archivado.
	PaymentStatusArchived PaymentStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusPaid, PaymentStatusRefunded,
		PaymentStatusForfeited, PaymentStatusArchived:
		return true
	}
	return false
}

// ReservationPayment representa un registro de pago vinculado a una
// reserva.
type ReservationPayment struct {
	ID                string
	ReservationID     string
	PaymentID         *string
	VoucherURL        *string
	Amount            float64
	IsSecurityDeposit bool
	PaidAt            *time.Time
	RefundedAt        *time.Time
	Status            PaymentStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}
