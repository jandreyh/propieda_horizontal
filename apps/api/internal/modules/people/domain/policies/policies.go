// Package policies contiene funciones puras de logica de negocio del
// modulo people: normalizacion y validacion de placas vehiculares.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// colombianPlateRe acepta los dos formatos vigentes de placa en Colombia:
//   - ABC123  (autos particulares y publicos): 3 letras + 3 digitos.
//   - ABC12A  (motocicletas): 3 letras + 2 digitos + 1 letra.
//
// La validacion se hace sobre la placa YA normalizada (uppercase + trim);
// los anchors aseguran longitud exacta.
var colombianPlateRe = regexp.MustCompile(`^[A-Z]{3}(?:[0-9]{3}|[0-9]{2}[A-Z])$`)

// NormalizePlate aplica trim de espacios y uppercase a una placa libre.
//
// Esta funcion NO valida formato; los usecases combinan NormalizePlate
// con IsValidColombianPlate para rechazar entradas mal formadas.
func NormalizePlate(plate string) string {
	return strings.ToUpper(strings.TrimSpace(plate))
}

// IsValidColombianPlate indica si `plate` (ya normalizada con
// NormalizePlate) corresponde a un formato vigente en Colombia:
// `ABC123` (autos) o `ABC12A` (motos).
func IsValidColombianPlate(plate string) bool {
	return colombianPlateRe.MatchString(plate)
}

// ValidatePlate normaliza la placa, valida su formato y devuelve la
// version normalizada lista para persistir. Retorna error si esta vacia
// o si no calza el formato Colombiano.
func ValidatePlate(plate string) (string, error) {
	p := NormalizePlate(plate)
	if p == "" {
		return "", errors.New("plate is required")
	}
	if len(p) > 16 {
		return "", fmt.Errorf("plate too long (max 16, got %d)", len(p))
	}
	if !IsValidColombianPlate(p) {
		return "", fmt.Errorf("invalid Colombian plate %q: expected ABC123 or ABC12A", p)
	}
	return p, nil
}

// ValidateVehicleType verifica que el tipo este en el set permitido. La
// lista vive en entities pero replicamos los strings aqui para mantener
// policies puras (sin importar entities y crear ciclo).
var allowedVehicleTypes = map[string]struct{}{
	"car":        {},
	"motorcycle": {},
	"truck":      {},
	"bicycle":    {},
	"other":      {},
}

// ValidateVehicleType verifica que el tipo este en el set permitido.
func ValidateVehicleType(t string) error {
	if t == "" {
		return errors.New("vehicle type is required")
	}
	if _, ok := allowedVehicleTypes[t]; !ok {
		return fmt.Errorf("invalid vehicle type %q (allowed: car, motorcycle, truck, bicycle, other)", t)
	}
	return nil
}

// ValidateVehicleYear acepta nil o un anio entre 1950 y 2100.
func ValidateVehicleYear(year *int32) error {
	if year == nil {
		return nil
	}
	if *year < 1950 || *year > 2100 {
		return fmt.Errorf("invalid year %d (allowed: 1950..2100)", *year)
	}
	return nil
}

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23). Util para fallar
// rapido en entradas HTTP antes de tocar la base.
func ValidateUUID(id string) error {
	if len(id) != 36 {
		return fmt.Errorf("invalid uuid length (expected 36, got %d)", len(id))
	}
	for i, c := range id {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return errors.New("invalid uuid format")
			}
		default:
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return errors.New("invalid uuid format")
			}
		}
	}
	return nil
}
