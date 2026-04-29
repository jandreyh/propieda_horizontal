package entities

import "time"

// BillingAccountStatus enumera los estados validos de una cuenta de
// facturacion.
type BillingAccountStatus string

// Possible values for BillingAccountStatus.
const (
	BillingAccountStatusActive   BillingAccountStatus = "active"
	BillingAccountStatusClosed   BillingAccountStatus = "closed"
	BillingAccountStatusArchived BillingAccountStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s BillingAccountStatus) IsValid() bool {
	switch s {
	case BillingAccountStatusActive, BillingAccountStatusClosed,
		BillingAccountStatusArchived:
		return true
	}
	return false
}

// BillingAccount representa la cuenta contrato entre una unidad
// inmobiliaria y su titular.
type BillingAccount struct {
	ID           string
	UnitID       string
	HolderUserID string
	OpenedAt     time.Time
	ClosedAt     *time.Time
	Status       BillingAccountStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}

// IsActive indica si la cuenta esta activa y no soft-deleted.
func (b BillingAccount) IsActive() bool {
	return b.Status == BillingAccountStatusActive && b.DeletedAt == nil
}
