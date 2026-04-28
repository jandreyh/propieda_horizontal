// Package policies agrupa funciones puras que codifican reglas de
// negocio del modulo units. No tienen efectos ni dependen de IO; son
// directamente testeables sin DB ni HTTP.
package policies

import (
	"github.com/saas-ph/api/internal/modules/units/domain/entities"
)

// MaxOwnershipPercentage es la suma maxima de porcentajes que pueden
// tener simultaneamente los propietarios activos de una unidad. Es la
// regla colombiana estandar de copropiedad.
const MaxOwnershipPercentage = 100.0

// ValidatePercentageRange verifica que el porcentaje individual de una
// fila este en el rango aceptado por la base (0, 100].
func ValidatePercentageRange(p float64) bool {
	return p > 0 && p <= MaxOwnershipPercentage
}

// SumActivePercentages calcula la suma de porcentajes de owners activos
// (los que aun no tienen until_date). Filtra defensivamente por
// IsActive para tolerar listas mixtas.
func SumActivePercentages(active []entities.UnitOwner) float64 {
	total := 0.0
	for i := range active {
		if !active[i].IsActive() {
			continue
		}
		total += active[i].Percentage
	}
	return total
}

// ValidatePercentage indica si agregar un nuevo porcentaje (newPct) a
// la lista de owners activos current dejaria la suma <= 100. Tambien
// verifica que el porcentaje individual sea valido.
//
// Esta es la regla "ValidatePercentage" del DoD: ninguna combinacion
// de owners activos puede exceder el 100% del coeficiente de la unidad.
func ValidatePercentage(current []entities.UnitOwner, newPct float64) bool {
	if !ValidatePercentageRange(newPct) {
		return false
	}
	total := SumActivePercentages(current) + newPct
	// Tolerancia minima por errores de redondeo en NUMERIC(5,2).
	const eps = 1e-9
	return total <= MaxOwnershipPercentage+eps
}

// EnsureOnlyOnePrimary indica si la lista de occupants activos
// satisface la invariante "a lo sumo un primary por unidad". Devuelve
// true cuando es valida.
//
// Si se esta evaluando antes de insertar un nuevo occupant primario,
// pasar isNewPrimary=true; el helper agrega ese 1 al conteo.
func EnsureOnlyOnePrimary(active []entities.UnitOccupancy, isNewPrimary bool) bool {
	count := 0
	for i := range active {
		if !active[i].IsActive() {
			continue
		}
		if active[i].IsPrimary {
			count++
		}
	}
	if isNewPrimary {
		count++
	}
	return count <= 1
}
