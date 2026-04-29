package entities

import "time"

// PenaltyStatus enumera los estados validos de una sancion.
type PenaltyStatus string

const (
	// PenaltyStatusDrafted indica que la sancion esta en borrador.
	PenaltyStatusDrafted PenaltyStatus = "drafted"
	// PenaltyStatusNotified indica que el deudor fue notificado.
	PenaltyStatusNotified PenaltyStatus = "notified"
	// PenaltyStatusInAppeal indica que hay una apelacion activa.
	PenaltyStatusInAppeal PenaltyStatus = "in_appeal"
	// PenaltyStatusConfirmed indica que la sancion fue confirmada.
	PenaltyStatusConfirmed PenaltyStatus = "confirmed"
	// PenaltyStatusSettled indica que la sancion fue saldada.
	PenaltyStatusSettled PenaltyStatus = "settled"
	// PenaltyStatusDismissed indica que la sancion fue desestimada.
	PenaltyStatusDismissed PenaltyStatus = "dismissed"
	// PenaltyStatusCancelled indica que la sancion fue cancelada.
	PenaltyStatusCancelled PenaltyStatus = "cancelled"
)

// IsValid indica si el status es uno de los enumerados.
func (s PenaltyStatus) IsValid() bool {
	switch s {
	case PenaltyStatusDrafted, PenaltyStatusNotified, PenaltyStatusInAppeal,
		PenaltyStatusConfirmed, PenaltyStatusSettled, PenaltyStatusDismissed,
		PenaltyStatusCancelled:
		return true
	}
	return false
}

// Penalty representa una sancion impuesta a un deudor.
type Penalty struct {
	ID                      string
	CatalogID               string
	DebtorUserID            string
	UnitID                  *string
	SourceIncidentID        *string
	SanctionType            SanctionType
	Amount                  float64
	Reason                  string
	ImposedByUserID         string
	NotifiedAt              *time.Time
	AppealDeadlineAt        *time.Time
	ConfirmedAt             *time.Time
	SettledAt               *time.Time
	DismissedAt             *time.Time
	CancelledAt             *time.Time
	RequiresCouncilApproval bool
	CouncilApprovedByUserID *string
	CouncilApprovedAt       *time.Time
	IdempotencyKey          *string
	Status                  PenaltyStatus
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               *time.Time
	CreatedBy               *string
	UpdatedBy               *string
	DeletedBy               *string
	Version                 int32
}
