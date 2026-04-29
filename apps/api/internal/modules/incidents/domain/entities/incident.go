// Package entities define las entidades de dominio del modulo incidents.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// IncidentType enumera los tipos validos de un incidente.
type IncidentType string

// Possible values for IncidentType.
const (
	IncidentTypeNoise        IncidentType = "noise"
	IncidentTypeLeak         IncidentType = "leak"
	IncidentTypeDamage       IncidentType = "damage"
	IncidentTypeTheftAttempt IncidentType = "theft_attempt"
	IncidentTypeAccident     IncidentType = "accident"
	IncidentTypePetIssue     IncidentType = "pet_issue"
	IncidentTypeOther        IncidentType = "other"
)

// IsValid indica si el tipo es uno de los enumerados.
func (t IncidentType) IsValid() bool {
	switch t {
	case IncidentTypeNoise, IncidentTypeLeak, IncidentTypeDamage,
		IncidentTypeTheftAttempt, IncidentTypeAccident,
		IncidentTypePetIssue, IncidentTypeOther:
		return true
	}
	return false
}

// Severity enumera los niveles de severidad de un incidente.
type Severity string

// Possible values for Severity.
const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// IsValid indica si la severidad es una de las enumeradas.
func (s Severity) IsValid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	}
	return false
}

// IncidentStatus enumera los estados validos de un incidente.
type IncidentStatus string

// Possible values for IncidentStatus.
const (
	IncidentStatusReported   IncidentStatus = "reported"
	IncidentStatusAssigned   IncidentStatus = "assigned"
	IncidentStatusInProgress IncidentStatus = "in_progress"
	IncidentStatusResolved   IncidentStatus = "resolved"
	IncidentStatusClosed     IncidentStatus = "closed"
	IncidentStatusCancelled  IncidentStatus = "cancelled"
)

// IsValid indica si el status es uno de los enumerados.
func (s IncidentStatus) IsValid() bool {
	switch s {
	case IncidentStatusReported, IncidentStatusAssigned,
		IncidentStatusInProgress, IncidentStatusResolved,
		IncidentStatusClosed, IncidentStatusCancelled:
		return true
	}
	return false
}

// Incident representa un incidente reportado en el conjunto residencial.
type Incident struct {
	ID               string
	IncidentType     IncidentType
	Severity         Severity
	Title            string
	Description      string
	ReportedByUserID string
	ReportedAt       time.Time
	StructureID      *string
	LocationDetail   *string
	AssignedToUserID *string
	AssignedAt       *time.Time
	StartedAt        *time.Time
	ResolvedAt       *time.Time
	ClosedAt         *time.Time
	CancelledAt      *time.Time
	ResolutionNotes  *string
	Escalated        bool
	SLAAssignDueAt   *time.Time
	SLAResolveDueAt  *time.Time
	Status           IncidentStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	DeletedBy        *string
	Version          int32
}

// IsActive indica si el incidente esta activo y no soft-deleted.
func (i Incident) IsActive() bool {
	return i.DeletedAt == nil
}

// IsTerminal indica si el incidente esta en un estado terminal.
func (i Incident) IsTerminal() bool {
	return i.Status == IncidentStatusClosed || i.Status == IncidentStatusCancelled
}
