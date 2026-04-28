// Package entities define las entidades de dominio del modulo
// access_control.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos; nunca aparece como
//     columna ni como campo de dominio.
package entities

import "time"

// DocumentType enumera los tipos de documento aceptados por el modulo
// (alineado con el set de identity).
type DocumentType string

const (
	// DocumentTypeCC es Cedula de Ciudadania (CO).
	DocumentTypeCC DocumentType = "CC"
	// DocumentTypeCE es Cedula de Extranjeria (CO).
	DocumentTypeCE DocumentType = "CE"
	// DocumentTypePA es Pasaporte.
	DocumentTypePA DocumentType = "PA"
	// DocumentTypeTI es Tarjeta de Identidad (menores).
	DocumentTypeTI DocumentType = "TI"
	// DocumentTypeRC es Registro Civil.
	DocumentTypeRC DocumentType = "RC"
	// DocumentTypeNIT es NIT (juridico).
	DocumentTypeNIT DocumentType = "NIT"
)

// IsValid indica si el tipo es uno de los enumerados validos.
func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentTypeCC, DocumentTypeCE, DocumentTypePA,
		DocumentTypeTI, DocumentTypeRC, DocumentTypeNIT:
		return true
	}
	return false
}

// BlacklistStatus enumera los estados de una entrada de blacklist.
type BlacklistStatus string

const (
	// BlacklistStatusActive marca la entrada como vigente.
	BlacklistStatusActive BlacklistStatus = "active"
	// BlacklistStatusArchived marca la entrada como soft-deleted.
	BlacklistStatusArchived BlacklistStatus = "archived"
)

// BlacklistEntry representa una persona vetada en porteria.
//
// La unicidad (un solo registro activo por documento) se enforza con un
// indice parcial unique en la base de datos.
type BlacklistEntry struct {
	ID               string
	DocumentType     DocumentType
	DocumentNumber   string
	FullName         *string
	Reason           string
	ReportedByUnitID *string
	ReportedByUserID *string
	ExpiresAt        *time.Time
	Status           BlacklistStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	DeletedBy        *string
	Version          int32
}

// IsArchived indica si la entrada esta soft-deleted.
func (b BlacklistEntry) IsArchived() bool {
	return b.Status == BlacklistStatusArchived || b.DeletedAt != nil
}
