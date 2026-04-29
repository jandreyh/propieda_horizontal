package entities

import "time"

// PeriodClosureStatus enumera los estados validos de un cierre de periodo.
type PeriodClosureStatus string

// Possible values for PeriodClosureStatus.
const (
	PeriodClosureStatusOpen       PeriodClosureStatus = "open"
	PeriodClosureStatusClosedSoft PeriodClosureStatus = "closed_soft"
	PeriodClosureStatusClosedHard PeriodClosureStatus = "closed_hard"
	PeriodClosureStatusArchived   PeriodClosureStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s PeriodClosureStatus) IsValid() bool {
	switch s {
	case PeriodClosureStatusOpen, PeriodClosureStatusClosedSoft,
		PeriodClosureStatusClosedHard, PeriodClosureStatusArchived:
		return true
	}
	return false
}

// PeriodClosure representa el cierre mensual de un periodo contable.
type PeriodClosure struct {
	ID           string
	PeriodYear   int32
	PeriodMonth  int32
	ClosedSoftAt *time.Time
	ClosedHardAt *time.Time
	ClosedBy     *string
	Notes        *string
	Status       PeriodClosureStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}
