// Package entities define las entidades de dominio del modulo parking.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// SpaceType enumera los tipos validos de un espacio de parqueadero.
type SpaceType string

const (
	// SpaceTypeCovered es un espacio cubierto.
	SpaceTypeCovered SpaceType = "covered"
	// SpaceTypeUncovered es un espacio descubierto (al aire libre).
	SpaceTypeUncovered SpaceType = "uncovered"
	// SpaceTypeMotorcycle es un espacio para motocicletas.
	SpaceTypeMotorcycle SpaceType = "motorcycle"
	// SpaceTypeBicycle es un espacio para bicicletas.
	SpaceTypeBicycle SpaceType = "bicycle"
	// SpaceTypeVisitor es un espacio reservado para visitantes.
	SpaceTypeVisitor SpaceType = "visitor"
	// SpaceTypeDisabled es un espacio para personas con movilidad reducida.
	SpaceTypeDisabled SpaceType = "disabled"
	// SpaceTypeElectric es un espacio con punto de carga electrica.
	SpaceTypeElectric SpaceType = "electric"
	// SpaceTypeDouble es un espacio doble (2 vehiculos).
	SpaceTypeDouble SpaceType = "double"
)

// IsValid indica si el tipo es uno de los enumerados.
func (t SpaceType) IsValid() bool {
	switch t {
	case SpaceTypeCovered, SpaceTypeUncovered, SpaceTypeMotorcycle,
		SpaceTypeBicycle, SpaceTypeVisitor, SpaceTypeDisabled,
		SpaceTypeElectric, SpaceTypeDouble:
		return true
	}
	return false
}

// SpaceStatus enumera los estados validos de un espacio de parqueadero.
type SpaceStatus string

const (
	// SpaceStatusActive indica que el espacio esta disponible para uso.
	SpaceStatusActive SpaceStatus = "active"
	// SpaceStatusInactive indica que el espacio esta deshabilitado
	// temporalmente.
	SpaceStatusInactive SpaceStatus = "inactive"
	// SpaceStatusMaintenance indica que el espacio esta en mantenimiento.
	SpaceStatusMaintenance SpaceStatus = "maintenance"
	// SpaceStatusArchived indica que el espacio fue archivado (soft delete
	// logico adicional).
	SpaceStatusArchived SpaceStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s SpaceStatus) IsValid() bool {
	switch s {
	case SpaceStatusActive, SpaceStatusInactive, SpaceStatusMaintenance,
		SpaceStatusArchived:
		return true
	}
	return false
}

// ParkingSpace representa un espacio fisico de parqueadero del conjunto
// residencial.
type ParkingSpace struct {
	ID          string
	Code        string
	Type        SpaceType
	StructureID *string
	Level       *string
	Zone        *string
	MonthlyFee  *float64
	IsVisitor   bool
	Notes       *string
	Status      SpaceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}

// IsActive indica si el espacio esta activo y no soft-deleted.
func (s ParkingSpace) IsActive() bool {
	return s.Status == SpaceStatusActive && s.DeletedAt == nil
}
