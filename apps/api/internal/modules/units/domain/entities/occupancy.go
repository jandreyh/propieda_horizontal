package entities

import "time"

// OccupancyRole es el rol funcional que cumple un usuario en una unidad
// concreta. Sincronizar con CHECK constraint de unit_occupancies.
type OccupancyRole string

// Roles permitidos.
const (
	OccupancyRoleOwnerResident OccupancyRole = "owner_resident"
	OccupancyRoleTenant        OccupancyRole = "tenant"
	OccupancyRoleAuthorized    OccupancyRole = "authorized"
	OccupancyRoleFamily        OccupancyRole = "family"
	OccupancyRoleStaff         OccupancyRole = "staff"
)

// IsValid indica si or es un rol permitido.
func (or OccupancyRole) IsValid() bool {
	switch or {
	case OccupancyRoleOwnerResident, OccupancyRoleTenant,
		OccupancyRoleAuthorized, OccupancyRoleFamily, OccupancyRoleStaff:
		return true
	}
	return false
}

// OccupancyStatus refleja el ciclo administrativo de la fila.
// La vigencia operativa la indica MoveOutDate.
type OccupancyStatus string

// Estados permitidos.
const (
	OccupancyStatusActive   OccupancyStatus = "active"
	OccupancyStatusInactive OccupancyStatus = "inactive"
	OccupancyStatusArchived OccupancyStatus = "archived"
)

// UnitOccupancy registra a un usuario que ocupa una unidad: el dueno
// que reside, el inquilino, autorizados, familiares y staff. Solo una
// fila por unidad puede tener IsPrimary=true mientras este vigente.
type UnitOccupancy struct {
	ID          string
	UnitID      string
	UserID      string
	RoleInUnit  OccupancyRole
	IsPrimary   bool
	MoveInDate  time.Time
	MoveOutDate *time.Time
	Status      OccupancyStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int
}

// IsActive devuelve true cuando la ocupacion esta vigente (no
// soft-deleted y sin fecha de salida).
func (o *UnitOccupancy) IsActive() bool {
	if o == nil {
		return false
	}
	if o.DeletedAt != nil {
		return false
	}
	return o.MoveOutDate == nil
}
