package entities

import "time"

// IncidentAssignmentStatus enumera los estados validos de una asignacion
// de incidente.
type IncidentAssignmentStatus string

const (
	// IncidentAssignmentStatusActive indica que la asignacion esta vigente.
	IncidentAssignmentStatusActive IncidentAssignmentStatus = "active"
	// IncidentAssignmentStatusUnassigned indica que la asignacion fue
	// desactivada.
	IncidentAssignmentStatusUnassigned IncidentAssignmentStatus = "unassigned"
)

// IsValid indica si el status es uno de los enumerados.
func (s IncidentAssignmentStatus) IsValid() bool {
	switch s {
	case IncidentAssignmentStatusActive, IncidentAssignmentStatusUnassigned:
		return true
	}
	return false
}

// IncidentAssignment representa la asignacion de un incidente a un usuario.
type IncidentAssignment struct {
	ID               string
	IncidentID       string
	AssignedToUserID string
	AssignedByUserID string
	AssignedAt       time.Time
	UnassignedAt     *time.Time
	Status           IncidentAssignmentStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	DeletedBy        *string
}
