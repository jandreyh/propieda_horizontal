package entities

import "time"

// PersonRoleInUnit unifica los roles posibles que devuelve la consulta
// "personas activas en una unidad", que mezcla owners y occupants.
// Para owners se usa el valor "owner"; para occupants se reusa el
// OccupancyRole correspondiente (owner_resident, tenant, etc.).
type PersonRoleInUnit string

// PersonRoleOwner es el rol asignado a las filas provenientes de
// unit_owners en GetActivePeopleForUnit.
const PersonRoleOwner PersonRoleInUnit = "owner"

// PersonInUnit es el resultado del JOIN owners + occupants para
// presentar "quien esta hoy en la unidad" en un solo listado.
//
// El campo IsPrimary solo aplica a occupants (true a lo sumo en uno
// por unidad). Para owners viene siempre false.
//
// SinceDate corresponde a since_date del owner o move_in_date del
// occupant, segun de donde provenga la fila.
type PersonInUnit struct {
	UserID     string
	FullName   string
	Document   string
	RoleInUnit PersonRoleInUnit
	IsPrimary  bool
	SinceDate  time.Time
}
