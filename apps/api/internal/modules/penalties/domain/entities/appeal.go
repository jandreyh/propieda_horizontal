package entities

import "time"

// AppealStatus enumera los estados validos de una apelacion.
type AppealStatus string

const (
	// AppealStatusSubmitted indica que la apelacion fue presentada.
	AppealStatusSubmitted AppealStatus = "submitted"
	// AppealStatusUnderReview indica que la apelacion esta en revision.
	AppealStatusUnderReview AppealStatus = "under_review"
	// AppealStatusAccepted indica que la apelacion fue aceptada.
	AppealStatusAccepted AppealStatus = "accepted"
	// AppealStatusRejected indica que la apelacion fue rechazada.
	AppealStatusRejected AppealStatus = "rejected"
	// AppealStatusWithdrawn indica que la apelacion fue retirada.
	AppealStatusWithdrawn AppealStatus = "withdrawn"
)

// IsValid indica si el status es uno de los enumerados.
func (s AppealStatus) IsValid() bool {
	switch s {
	case AppealStatusSubmitted, AppealStatusUnderReview,
		AppealStatusAccepted, AppealStatusRejected,
		AppealStatusWithdrawn:
		return true
	}
	return false
}

// IsTerminal indica si el status es terminal (no permite mas transiciones).
func (s AppealStatus) IsTerminal() bool {
	switch s {
	case AppealStatusAccepted, AppealStatusRejected, AppealStatusWithdrawn:
		return true
	}
	return false
}

// PenaltyAppeal representa una apelacion de un residente a una sancion.
type PenaltyAppeal struct {
	ID                string
	PenaltyID         string
	SubmittedByUserID string
	SubmittedAt       time.Time
	Grounds           string
	ResolvedByUserID  *string
	ResolvedAt        *time.Time
	Resolution        *string
	Status            AppealStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}
