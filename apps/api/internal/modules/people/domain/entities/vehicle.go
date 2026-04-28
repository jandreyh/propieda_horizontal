// Package entities define las entidades de dominio del modulo people.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos; nunca aparece como
//     columna ni como campo de dominio.
package entities

import "time"

// VehicleType enumera los tipos validos de un vehiculo.
type VehicleType string

const (
	// VehicleTypeCar es un automovil.
	VehicleTypeCar VehicleType = "car"
	// VehicleTypeMotorcycle es una motocicleta.
	VehicleTypeMotorcycle VehicleType = "motorcycle"
	// VehicleTypeTruck es una camioneta / camion.
	VehicleTypeTruck VehicleType = "truck"
	// VehicleTypeBicycle es una bicicleta.
	VehicleTypeBicycle VehicleType = "bicycle"
	// VehicleTypeOther es cualquier otro vehiculo.
	VehicleTypeOther VehicleType = "other"
)

// IsValid indica si el tipo es uno de los enumerados validos.
func (t VehicleType) IsValid() bool {
	switch t {
	case VehicleTypeCar, VehicleTypeMotorcycle, VehicleTypeTruck,
		VehicleTypeBicycle, VehicleTypeOther:
		return true
	}
	return false
}

// VehicleStatus enumera los estados validos de un Vehicle.
type VehicleStatus string

const (
	// VehicleStatusActive marca el vehiculo como vigente.
	VehicleStatusActive VehicleStatus = "active"
	// VehicleStatusInactive marca el vehiculo como inactivo (no vigente).
	VehicleStatusInactive VehicleStatus = "inactive"
	// VehicleStatusArchived marca el vehiculo como soft-deleted.
	VehicleStatusArchived VehicleStatus = "archived"
)

// Vehicle representa un vehiculo registrado en el conjunto. La placa se
// guarda ya normalizada (uppercase + trim) por la capa de aplicacion.
type Vehicle struct {
	ID        string
	Plate     string
	Type      VehicleType
	Brand     *string
	Model     *string
	Color     *string
	Year      *int32
	Status    VehicleStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int32
}

// IsArchived indica si el vehiculo esta soft-deleted.
func (v Vehicle) IsArchived() bool {
	return v.Status == VehicleStatusArchived || v.DeletedAt != nil
}
