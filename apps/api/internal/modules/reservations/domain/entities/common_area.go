// Package entities define las entidades de dominio del modulo reservations.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// CommonAreaKind enumera los tipos validos de una zona comun.
type CommonAreaKind string

const (
	// CommonAreaKindSalonSocial es un salon social.
	CommonAreaKindSalonSocial CommonAreaKind = "salon_social"
	// CommonAreaKindBBQ es una zona de barbacoa.
	CommonAreaKindBBQ CommonAreaKind = "bbq"
	// CommonAreaKindPiscina es una piscina.
	CommonAreaKindPiscina CommonAreaKind = "piscina"
	// CommonAreaKindGym es un gimnasio.
	CommonAreaKindGym CommonAreaKind = "gym"
	// CommonAreaKindCancha es una cancha deportiva.
	CommonAreaKindCancha CommonAreaKind = "cancha"
	// CommonAreaKindSalaEstudio es una sala de estudio.
	CommonAreaKindSalaEstudio CommonAreaKind = "sala_estudio"
	// CommonAreaKindOther es otro tipo de zona comun.
	CommonAreaKindOther CommonAreaKind = "other"
)

// IsValid indica si el kind es uno de los enumerados.
func (k CommonAreaKind) IsValid() bool {
	switch k {
	case CommonAreaKindSalonSocial, CommonAreaKindBBQ, CommonAreaKindPiscina,
		CommonAreaKindGym, CommonAreaKindCancha, CommonAreaKindSalaEstudio,
		CommonAreaKindOther:
		return true
	}
	return false
}

// CommonAreaStatus enumera los estados validos de una zona comun.
type CommonAreaStatus string

const (
	// CommonAreaStatusActive indica que la zona esta habilitada.
	CommonAreaStatusActive CommonAreaStatus = "active"
	// CommonAreaStatusInactive indica que la zona esta deshabilitada.
	CommonAreaStatusInactive CommonAreaStatus = "inactive"
	// CommonAreaStatusArchived indica que la zona fue archivada.
	CommonAreaStatusArchived CommonAreaStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s CommonAreaStatus) IsValid() bool {
	switch s {
	case CommonAreaStatusActive, CommonAreaStatusInactive, CommonAreaStatusArchived:
		return true
	}
	return false
}

// CommonArea representa una zona comun del conjunto residencial.
type CommonArea struct {
	ID                  string
	Code                string
	Name                string
	Kind                CommonAreaKind
	MaxCapacity         *int32
	OpeningTime         *string
	ClosingTime         *string
	SlotDurationMinutes int32
	CostPerUse          float64
	SecurityDeposit     float64
	RequiresApproval    bool
	IsActive            bool
	Description         *string
	Status              CommonAreaStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	CreatedBy           *string
	UpdatedBy           *string
	DeletedBy           *string
	Version             int32
}

// IsAvailable indica si la zona comun esta activa y no soft-deleted.
func (c CommonArea) IsAvailable() bool {
	return c.Status == CommonAreaStatusActive && c.IsActive && c.DeletedAt == nil
}
