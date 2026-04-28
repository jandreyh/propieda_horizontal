package entities

import "time"

// Permission representa una entrada del catalogo estatico de permisos
// del producto. Se inyecta en cada Tenant DB via migracion seed; los
// tenants NO crean permisos nuevos.
type Permission struct {
	ID          string
	Namespace   string // ej. "package.deliver"
	Description string
	Status      string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
}
