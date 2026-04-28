package entities

import "time"

// OwnerStatus refleja el ciclo de vida administrativo de la fila.
// La regla operativa (vigencia del propietario) la indica UntilDate.
type OwnerStatus string

// Estados permitidos. Sincronizar con CHECK constraint.
const (
	OwnerStatusActive   OwnerStatus = "active"
	OwnerStatusInactive OwnerStatus = "inactive"
	OwnerStatusArchived OwnerStatus = "archived"
)

// UnitOwner es la asociacion historica entre un usuario y una unidad
// como propietario, con su porcentaje de propiedad. Una venta cierra
// la fila vigente seteando UntilDate; una compra crea una fila nueva.
//
// La unicidad operativa (un mismo usuario solo puede ser propietario
// activo una vez por unidad) la garantiza el INDEX UNIQUE parcial sobre
// (unit_id, user_id) WHERE deleted_at IS NULL AND until_date IS NULL.
type UnitOwner struct {
	ID         string
	UnitID     string
	UserID     string
	Percentage float64
	SinceDate  time.Time
	UntilDate  *time.Time
	Status     OwnerStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	CreatedBy  *string
	UpdatedBy  *string
	DeletedBy  *string
	Version    int
}

// IsActive devuelve true cuando la fila representa una propiedad
// vigente (no soft-deleted y sin until_date establecida).
func (o *UnitOwner) IsActive() bool {
	if o == nil {
		return false
	}
	if o.DeletedAt != nil {
		return false
	}
	return o.UntilDate == nil
}
