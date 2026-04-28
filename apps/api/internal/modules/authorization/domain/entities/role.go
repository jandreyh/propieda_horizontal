// Package entities contiene las entidades de dominio del modulo
// authorization. Son structs puros: NO conocen JSON ni DB.
package entities

import "time"

// Role representa un rol del producto (semilla, IsSystem=true) o un rol
// custom creado por un tenant_admin.
//
// Reglas:
//   - Si IsSystem es true, el rol no puede ser editado ni eliminado por
//     la API de gestion (solo via migracion de plataforma).
//   - Status pertenece a {"active", "archived"}.
type Role struct {
	ID          string
	Name        string
	Description string
	IsSystem    bool
	Status      string
	Permissions []Permission // opcional; se popula con GetRoleByID

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int
}

// IsActive indica si el rol esta activo (no archivado, no soft-deleted).
func (r Role) IsActive() bool {
	return r.Status == StatusActive && r.DeletedAt == nil
}

// Constantes de status para Role.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)
