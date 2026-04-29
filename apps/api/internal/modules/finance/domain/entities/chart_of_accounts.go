// Package entities define las entidades de dominio del modulo finance.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// AccountType enumera los tipos validos de cuenta contable.
type AccountType string

// Possible values for AccountType.
const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeIncome    AccountType = "income"
	AccountTypeExpense   AccountType = "expense"
)

// IsValid indica si el tipo es uno de los enumerados.
func (t AccountType) IsValid() bool {
	switch t {
	case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity,
		AccountTypeIncome, AccountTypeExpense:
		return true
	}
	return false
}

// AccountStatus enumera los estados validos de una cuenta contable.
type AccountStatus string

// Possible values for AccountStatus.
const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusInactive AccountStatus = "inactive"
	AccountStatusArchived AccountStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s AccountStatus) IsValid() bool {
	switch s {
	case AccountStatusActive, AccountStatusInactive, AccountStatusArchived:
		return true
	}
	return false
}

// ChartOfAccount representa una cuenta del plan de cuentas.
type ChartOfAccount struct {
	ID          string
	Code        string
	Name        string
	AccountType AccountType
	ParentID    *string
	Status      AccountStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}

// IsActive indica si la cuenta esta activa y no soft-deleted.
func (a ChartOfAccount) IsActive() bool {
	return a.Status == AccountStatusActive && a.DeletedAt == nil
}
