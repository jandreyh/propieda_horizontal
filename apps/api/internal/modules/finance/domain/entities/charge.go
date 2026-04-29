package entities

import "time"

// ChargeConcept enumera los tipos de concepto de cargo.
type ChargeConcept string

// Possible values for ChargeConcept.
const (
	ChargeConceptAdminFee ChargeConcept = "admin_fee"
	ChargeConceptLateFee  ChargeConcept = "late_fee"
	ChargeConceptInterest ChargeConcept = "interest"
	ChargeConceptService  ChargeConcept = "service"
	ChargeConceptRental   ChargeConcept = "rental"
	ChargeConceptPenalty  ChargeConcept = "penalty"
	ChargeConceptOther    ChargeConcept = "other"
)

// IsValid indica si el concepto es uno de los enumerados.
func (c ChargeConcept) IsValid() bool {
	switch c {
	case ChargeConceptAdminFee, ChargeConceptLateFee, ChargeConceptInterest,
		ChargeConceptService, ChargeConceptRental, ChargeConceptPenalty,
		ChargeConceptOther:
		return true
	}
	return false
}

// ChargeStatus enumera los estados validos de un cargo.
type ChargeStatus string

// Possible values for ChargeStatus.
const (
	ChargeStatusOpen     ChargeStatus = "open"
	ChargeStatusPartial  ChargeStatus = "partial"
	ChargeStatusPaid     ChargeStatus = "paid"
	ChargeStatusVoided   ChargeStatus = "voided"
	ChargeStatusArchived ChargeStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s ChargeStatus) IsValid() bool {
	switch s {
	case ChargeStatusOpen, ChargeStatusPartial, ChargeStatusPaid,
		ChargeStatusVoided, ChargeStatusArchived:
		return true
	}
	return false
}

// Charge representa un cargo (cuota, multa, interes, servicio, etc.)
// contra una cuenta de facturacion.
type Charge struct {
	ID               string
	BillingAccountID string
	Concept          ChargeConcept
	PeriodYear       *int32
	PeriodMonth      *int32
	Amount           float64
	Balance          float64
	DueDate          time.Time
	CostCenterID     *string
	AccountID        *string
	IdempotencyKey   *string
	Description      *string
	Status           ChargeStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	DeletedBy        *string
	Version          int32
}

// IsPending indica si el cargo tiene saldo pendiente.
func (c Charge) IsPending() bool {
	return c.Balance > 0 && c.DeletedAt == nil &&
		(c.Status == ChargeStatusOpen || c.Status == ChargeStatusPartial)
}

// ChargeItem representa una linea de detalle de un cargo.
type ChargeItem struct {
	ID          string
	ChargeID    string
	Description string
	Amount      float64
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}
