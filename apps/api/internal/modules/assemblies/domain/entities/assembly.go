// Package entities define las entidades de dominio del modulo assemblies.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// AssemblyType enumera los tipos validos de una asamblea.
type AssemblyType string

const (
	// AssemblyTypeOrdinaria es la asamblea ordinaria (anual).
	AssemblyTypeOrdinaria AssemblyType = "ordinaria"
	// AssemblyTypeExtraordinaria es convocada para temas urgentes.
	AssemblyTypeExtraordinaria AssemblyType = "extraordinaria"
	// AssemblyTypeVirtual se realiza completamente en linea.
	AssemblyTypeVirtual AssemblyType = "virtual"
	// AssemblyTypeMixta combina presencial y virtual.
	AssemblyTypeMixta AssemblyType = "mixta"
)

// IsValid indica si el tipo es uno de los enumerados.
func (t AssemblyType) IsValid() bool {
	switch t {
	case AssemblyTypeOrdinaria, AssemblyTypeExtraordinaria,
		AssemblyTypeVirtual, AssemblyTypeMixta:
		return true
	}
	return false
}

// VotingMode enumera los modos de votacion de una asamblea.
type VotingMode string

const (
	// VotingModeCoefficient pondera los votos por coeficiente de
	// copropiedad.
	VotingModeCoefficient VotingMode = "coefficient"
	// VotingModeOneUnitOneVote asigna un voto por unidad.
	VotingModeOneUnitOneVote VotingMode = "one_unit_one_vote"
)

// IsValid indica si el modo de votacion es valido.
func (m VotingMode) IsValid() bool {
	switch m {
	case VotingModeCoefficient, VotingModeOneUnitOneVote:
		return true
	}
	return false
}

// AssemblyStatus enumera los estados validos de una asamblea.
type AssemblyStatus string

const (
	// AssemblyStatusDraft la asamblea esta en borrador.
	AssemblyStatusDraft AssemblyStatus = "draft"
	// AssemblyStatusCalled la asamblea fue convocada formalmente.
	AssemblyStatusCalled AssemblyStatus = "called"
	// AssemblyStatusInProgress la asamblea esta en curso.
	AssemblyStatusInProgress AssemblyStatus = "in_progress"
	// AssemblyStatusClosed la asamblea finalizo exitosamente.
	AssemblyStatusClosed AssemblyStatus = "closed"
	// AssemblyStatusQuorumFailed la asamblea no alcanzo quorum.
	AssemblyStatusQuorumFailed AssemblyStatus = "quorum_failed"
	// AssemblyStatusArchived la asamblea fue archivada.
	AssemblyStatusArchived AssemblyStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s AssemblyStatus) IsValid() bool {
	switch s {
	case AssemblyStatusDraft, AssemblyStatusCalled, AssemblyStatusInProgress,
		AssemblyStatusClosed, AssemblyStatusQuorumFailed, AssemblyStatusArchived:
		return true
	}
	return false
}

// Assembly representa una asamblea del conjunto residencial.
type Assembly struct {
	ID                string
	Name              string
	AssemblyType      AssemblyType
	ScheduledAt       time.Time
	VotingMode        VotingMode
	QuorumRequiredPct float64
	Location          *string
	Notes             *string
	StartedAt         *time.Time
	ClosedAt          *time.Time
	Status            AssemblyStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	CreatedBy         *string
	UpdatedBy         *string
	DeletedBy         *string
	Version           int32
}

// IsActive indica si la asamblea esta activa y no soft-deleted.
func (a Assembly) IsActive() bool {
	return a.DeletedAt == nil
}
