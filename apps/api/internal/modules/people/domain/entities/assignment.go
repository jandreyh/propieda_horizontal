package entities

import "time"

// AssignmentStatus enumera los estados validos de UnitVehicleAssignment.
type AssignmentStatus string

const (
	// AssignmentStatusActive marca la asignacion como vigente.
	AssignmentStatusActive AssignmentStatus = "active"
	// AssignmentStatusInactive marca la asignacion como inactiva (cerrada
	// con until_date pero no archivada).
	AssignmentStatusInactive AssignmentStatus = "inactive"
	// AssignmentStatusArchived marca la asignacion como soft-deleted.
	AssignmentStatusArchived AssignmentStatus = "archived"
)

// UnitVehicleAssignment representa la relacion historica de un vehiculo
// con una unidad (apartamento). Un vehiculo activo (until_date = nil) no
// puede estar asignado a mas de una unidad simultaneamente; la regla se
// enforza con un UNIQUE INDEX parcial en la base.
type UnitVehicleAssignment struct {
	ID        string
	UnitID    string
	VehicleID string
	SinceDate time.Time
	UntilDate *time.Time
	Status    AssignmentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int32

	// Vehicle es opcional: cuando el repo materializa el join con vehicles
	// (caso ListActiveByUnit) se rellena. En otros caminos puede ser nil.
	Vehicle *Vehicle
}

// IsActive indica si la asignacion sigue vigente (sin fecha de cierre).
func (a UnitVehicleAssignment) IsActive() bool {
	return a.UntilDate == nil && a.DeletedAt == nil
}
