// Package entities contiene las entidades puras del dominio units.
//
// Estas estructuras NO llevan tags JSON ni DB. Se mapean explicitamente a
// DTOs (capa application) y a structs de sqlc (capa infrastructure).
package entities

import "time"

// UnitType clasifica fisicamente la unidad operativa.
type UnitType string

// Tipos permitidos. Sincronizar con CHECK constraint de la tabla units.
const (
	UnitTypeApartment  UnitType = "apartment"
	UnitTypeHouse      UnitType = "house"
	UnitTypeCommercial UnitType = "commercial"
	UnitTypeOffice     UnitType = "office"
	UnitTypeParking    UnitType = "parking"
	UnitTypeStorage    UnitType = "storage"
	UnitTypeOther      UnitType = "other"
)

// IsValid indica si ut es uno de los valores aceptados.
func (ut UnitType) IsValid() bool {
	switch ut {
	case UnitTypeApartment, UnitTypeHouse, UnitTypeCommercial,
		UnitTypeOffice, UnitTypeParking, UnitTypeStorage, UnitTypeOther:
		return true
	}
	return false
}

// UnitStatus controla la disponibilidad operativa de la unidad.
type UnitStatus string

// Estados permitidos. Sincronizar con CHECK constraint.
const (
	UnitStatusActive   UnitStatus = "active"
	UnitStatusInactive UnitStatus = "inactive"
	UnitStatusArchived UnitStatus = "archived"
)

// Unit representa una unidad fisica del conjunto: apartamento, casa,
// local, oficina, parqueadero o deposito. Es la entidad raiz del
// agregado units (owners y occupants pivotan sobre el id de Unit).
//
// StructureID es opcional: en conjuntos sin torres declaradas, las
// unidades se cuelgan directo del tenant.
type Unit struct {
	ID          string
	StructureID *string
	Code        string
	Type        UnitType
	AreaM2      *float64
	Bedrooms    *int
	Coefficient *float64
	Status      UnitStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int
}

// IsActive devuelve true cuando la unidad esta operativa (no soft-deleted
// y status active).
func (u *Unit) IsActive() bool {
	if u == nil {
		return false
	}
	if u.DeletedAt != nil {
		return false
	}
	return u.Status == UnitStatusActive
}
