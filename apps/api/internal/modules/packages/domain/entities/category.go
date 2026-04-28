package entities

import "time"

// PackageCategory categoriza un paquete (sobre, caja, refrigerado).
//
// Algunas categorias requieren evidencia visual (foto) al recibir el
// paquete: por ejemplo, "Refrigerado".
type PackageCategory struct {
	ID               string
	Name             string
	RequiresEvidence bool
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	DeletedBy        *string
}
