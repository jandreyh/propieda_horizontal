package entities

import "time"

// DecisionType enumera los tipos de decision de una mocion.
type DecisionType string

const (
	// DecisionTypeSimple mayoria simple (>50%).
	DecisionTypeSimple DecisionType = "simple"
	// DecisionTypeQualified mayoria calificada (>70%).
	DecisionTypeQualified DecisionType = "qualified"
	// DecisionTypeSpecial supermayoria o unanimidad segun reglamento.
	DecisionTypeSpecial DecisionType = "special"
)

// IsValid indica si el tipo de decision es valido.
func (d DecisionType) IsValid() bool {
	switch d {
	case DecisionTypeSimple, DecisionTypeQualified, DecisionTypeSpecial:
		return true
	}
	return false
}

// VotingMethod enumera los metodos de votacion.
type VotingMethod string

const (
	// VotingMethodSecret votacion secreta.
	VotingMethodSecret VotingMethod = "secret"
	// VotingMethodNominal votacion nominal (publica).
	VotingMethodNominal VotingMethod = "nominal"
)

// IsValid indica si el metodo de votacion es valido.
func (m VotingMethod) IsValid() bool {
	switch m {
	case VotingMethodSecret, VotingMethodNominal:
		return true
	}
	return false
}

// MotionStatus enumera los estados validos de una mocion.
type MotionStatus string

const (
	// MotionStatusDraft la mocion esta en borrador.
	MotionStatusDraft MotionStatus = "draft"
	// MotionStatusOpen la votacion esta abierta.
	MotionStatusOpen MotionStatus = "open"
	// MotionStatusClosed la votacion fue cerrada.
	MotionStatusClosed MotionStatus = "closed"
	// MotionStatusCancelled la mocion fue cancelada.
	MotionStatusCancelled MotionStatus = "cancelled"
	// MotionStatusArchived la mocion fue archivada.
	MotionStatusArchived MotionStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s MotionStatus) IsValid() bool {
	switch s {
	case MotionStatusDraft, MotionStatusOpen, MotionStatusClosed,
		MotionStatusCancelled, MotionStatusArchived:
		return true
	}
	return false
}

// AssemblyMotion representa una mocion o punto de votacion dentro de
// una asamblea.
type AssemblyMotion struct {
	ID           string
	AssemblyID   string
	Title        string
	Description  *string
	DecisionType DecisionType
	VotingMethod VotingMethod
	Options      []byte
	OpensAt      *time.Time
	ClosesAt     *time.Time
	Results      []byte
	Status       MotionStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}
