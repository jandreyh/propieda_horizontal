package entities

import "time"

// AttendanceStatus enumera los estados validos de una asistencia.
type AttendanceStatus string

const (
	// AttendanceStatusPresent el asistente esta presente.
	AttendanceStatusPresent AttendanceStatus = "present"
	// AttendanceStatusLeft el asistente se retiro.
	AttendanceStatusLeft AttendanceStatus = "left"
	// AttendanceStatusVoiceOnly el asistente tiene voz pero no voto.
	AttendanceStatusVoiceOnly AttendanceStatus = "voice_only"
	// AttendanceStatusArchived la asistencia fue archivada.
	AttendanceStatusArchived AttendanceStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s AttendanceStatus) IsValid() bool {
	switch s {
	case AttendanceStatusPresent, AttendanceStatusLeft,
		AttendanceStatusVoiceOnly, AttendanceStatusArchived:
		return true
	}
	return false
}

// AssemblyAttendance representa el registro de asistencia de una unidad
// en una asamblea, incluyendo el coeficiente vigente en el momento del
// evento.
type AssemblyAttendance struct {
	ID                  string
	AssemblyID          string
	UnitID              string
	AttendeeUserID      *string
	RepresentedByUserID *string
	CoefficientAtEvent  float64
	ArrivalAt           time.Time
	DepartureAt         *time.Time
	IsRemote            bool
	HasVotingRight      bool
	Notes               *string
	Status              AttendanceStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	CreatedBy           *string
	UpdatedBy           *string
	DeletedBy           *string
	Version             int32
}
