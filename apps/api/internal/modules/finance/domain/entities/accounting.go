package entities

import "time"

// AccountingEntryStatus enumera los estados validos de un asiento contable.
type AccountingEntryStatus string

// Possible values for AccountingEntryStatus.
const (
	AccountingEntryStatusDraft    AccountingEntryStatus = "draft"
	AccountingEntryStatusPosted   AccountingEntryStatus = "posted"
	AccountingEntryStatusReversed AccountingEntryStatus = "reversed"
	AccountingEntryStatusArchived AccountingEntryStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s AccountingEntryStatus) IsValid() bool {
	switch s {
	case AccountingEntryStatusDraft, AccountingEntryStatusPosted,
		AccountingEntryStatusReversed, AccountingEntryStatusArchived:
		return true
	}
	return false
}

// AccountingEntry representa un asiento contable. Cuando sealed=true,
// el trigger de la DB impide modificaciones.
type AccountingEntry struct {
	ID          string
	PeriodYear  int32
	PeriodMonth int32
	PostedAt    time.Time
	SourceType  string
	SourceID    string
	Description *string
	Posted      bool
	Sealed      bool
	SealedAt    *time.Time
	Status      AccountingEntryStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}

// AccountingEntryLine representa una linea de un asiento contable.
type AccountingEntryLine struct {
	ID           string
	EntryID      string
	AccountID    string
	CostCenterID *string
	Debit        float64
	Credit       float64
	Description  *string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}
