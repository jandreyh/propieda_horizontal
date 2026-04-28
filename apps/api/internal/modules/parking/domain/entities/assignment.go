package entities

import "time"

// AssignmentStatus enumera los estados validos de una asignacion de
// parqueadero.
type AssignmentStatus string

const (
	// AssignmentStatusActive indica que la asignacion esta vigente.
	AssignmentStatusActive AssignmentStatus = "active"
	// AssignmentStatusClosed indica que la asignacion fue cerrada
	// (until_date establecido).
	AssignmentStatusClosed AssignmentStatus = "closed"
	// AssignmentStatusArchived indica que la asignacion fue archivada.
	AssignmentStatusArchived AssignmentStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s AssignmentStatus) IsValid() bool {
	switch s {
	case AssignmentStatusActive, AssignmentStatusClosed, AssignmentStatusArchived:
		return true
	}
	return false
}

// ParkingAssignment representa la asignacion vigente o historica de un
// espacio de parqueadero a una unidad inmobiliaria.
type ParkingAssignment struct {
	ID               string
	ParkingSpaceID   string
	UnitID           string
	VehicleID        *string
	AssignedByUserID *string
	SinceDate        time.Time
	UntilDate        *time.Time
	Notes            *string
	Status           AssignmentStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	DeletedBy        *string
	Version          int32
}

// AssignmentHistory representa un registro append-only de snapshot de
// reasignaciones para auditoria.
type AssignmentHistory struct {
	ID              string
	ParkingSpaceID  string
	UnitID          string
	AssignmentID    *string
	SinceDate       time.Time
	UntilDate       *time.Time
	ClosedReason    *string
	SnapshotPayload []byte
	RecordedAt      time.Time
	RecordedBy      *string
}
