// Package entities define las entidades de dominio del modulo penalties.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// SanctionType enumera los tipos validos de sancion.
type SanctionType string

const (
	// SanctionTypeWarning es una amonestacion verbal/escrita sin cargo.
	SanctionTypeWarning SanctionType = "warning"
	// SanctionTypeMonetary es una sancion pecuniaria con monto.
	SanctionTypeMonetary SanctionType = "monetary"
	// SanctionTypeServiceSuspension es una suspension de servicios del
	// conjunto.
	SanctionTypeServiceSuspension SanctionType = "service_suspension"
)

// IsValid indica si el tipo de sancion es uno de los enumerados.
func (t SanctionType) IsValid() bool {
	switch t {
	case SanctionTypeWarning, SanctionTypeMonetary, SanctionTypeServiceSuspension:
		return true
	}
	return false
}

// CatalogStatus enumera los estados del catalogo.
type CatalogStatus string

const (
	// CatalogStatusActive indica que la entrada del catalogo esta activa.
	CatalogStatusActive CatalogStatus = "active"
	// CatalogStatusArchived indica que la entrada fue archivada.
	CatalogStatusArchived CatalogStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s CatalogStatus) IsValid() bool {
	switch s {
	case CatalogStatusActive, CatalogStatusArchived:
		return true
	}
	return false
}

// PenaltyCatalog representa una entrada en el catalogo configurable de
// sanciones del tenant.
type PenaltyCatalog struct {
	ID                       string
	Code                     string
	Name                     string
	Description              *string
	DefaultSanctionType      SanctionType
	BaseAmount               float64
	RecurrenceMultiplier     float64
	RecurrenceCAPMultiplier  float64
	RequiresCouncilThreshold *float64
	Status                   CatalogStatus
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                *time.Time
	CreatedBy                *string
	UpdatedBy                *string
	DeletedBy                *string
	Version                  int32
}
