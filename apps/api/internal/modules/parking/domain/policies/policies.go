// Package policies contiene funciones puras de logica de negocio del
// modulo parking.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/parking/domain/entities"
)

// CanAssignSpace valida que un espacio pueda recibir una asignacion
// permanente (no visitante).
//
// Reglas:
//   - El espacio debe estar activo y no soft-deleted.
//   - El espacio NO debe ser de tipo visitante (is_visitor = false).
func CanAssignSpace(space entities.ParkingSpace) error {
	if !space.IsActive() {
		return errors.New("space is not active")
	}
	if space.IsVisitor {
		return errors.New("visitor spaces cannot be permanently assigned")
	}
	return nil
}

// CanReserveForVisitor valida que un espacio pueda recibir una reserva
// de visitante.
//
// Reglas:
//   - El espacio debe estar activo y no soft-deleted.
//   - El espacio debe ser de visitante (is_visitor = true) o de tipo
//     'visitor'.
func CanReserveForVisitor(space entities.ParkingSpace) error {
	if !space.IsActive() {
		return errors.New("space is not active")
	}
	if !space.IsVisitor && space.Type != entities.SpaceTypeVisitor {
		return errors.New("space is not designated for visitors")
	}
	return nil
}

// ValidateSlotDuration valida que la duracion de un slot de reserva no
// exceda el maximo permitido.
func ValidateSlotDuration(start, end time.Time, maxHours int) error {
	if !end.After(start) {
		return errors.New("slot end must be after slot start")
	}
	duration := end.Sub(start)
	maxDuration := time.Duration(maxHours) * time.Hour
	if duration > maxDuration {
		return fmt.Errorf("slot duration %v exceeds maximum of %d hours", duration, maxHours)
	}
	return nil
}

// ValidateReservationAdvance valida que la reserva se haga con la
// antelacion requerida (ni demasiado pronto ni demasiado tarde).
//
//   - minMinutes: minimo de minutos de antelacion antes del slot_start_at.
//   - maxDays: maximo de dias de antelacion antes del slot_start_at.
func ValidateReservationAdvance(start time.Time, now time.Time, minMinutes, maxDays int) error {
	advance := start.Sub(now)
	if advance < time.Duration(minMinutes)*time.Minute {
		return fmt.Errorf("reservation must be made at least %d minutes in advance", minMinutes)
	}
	maxAdvance := time.Duration(maxDays) * 24 * time.Hour
	if advance > maxAdvance {
		return fmt.Errorf("reservation cannot be made more than %d days in advance", maxDays)
	}
	return nil
}

// ShuffleDeterministic ejecuta un shuffle determinista de unit IDs
// usando SHA-256 como fuente de entropia reproducible.
//
// Algoritmo:
//  1. Ordenar unitIDs lexicograficamente.
//  2. Concatenar: seed + "|" + sorted_ids joined by ",".
//  3. Calcular SHA-256 del string resultante.
//  4. Usar los bytes del hash como semilla para Fisher-Yates shuffle.
//
// El resultado es reproducible dado el mismo seed y los mismos unitIDs.
func ShuffleDeterministic(seed string, unitIDs []string) []string {
	if len(unitIDs) <= 1 {
		result := make([]string, len(unitIDs))
		copy(result, unitIDs)
		return result
	}

	// 1. Copia y ordena lexicograficamente.
	sorted := make([]string, len(unitIDs))
	copy(sorted, unitIDs)
	sort.Strings(sorted)

	// 2. Construye el input del hash.
	joined := strings.Join(sorted, ",")
	hashInput := seed + "|" + joined

	// 3. SHA-256.
	hash := sha256.Sum256([]byte(hashInput))

	// 4. Fisher-Yates shuffle usando bytes del hash como entropia.
	// Si necesitamos mas bytes de los que el hash provee (32 bytes),
	// rehash con el hash anterior como input.
	result := make([]string, len(sorted))
	copy(result, sorted)

	hashBytes := hash[:]
	byteIdx := 0

	for i := len(result) - 1; i > 0; i-- {
		// Si nos quedamos sin bytes, rehash.
		if byteIdx+4 > len(hashBytes) {
			newHash := sha256.Sum256(hashBytes)
			hashBytes = newHash[:]
			byteIdx = 0
		}

		// Extraer un uint32 de 4 bytes para el indice.
		rnd := binary.BigEndian.Uint32(hashBytes[byteIdx : byteIdx+4])
		byteIdx += 4

		j := int(rnd % uint32(i+1)) //nolint:gosec // i is always positive (loop starts at len-1 and ends at 1)
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// CanTransitionReservation indica si una transicion de status de reserva
// es legal.
//
// Transiciones permitidas:
//   - pending    -> confirmed, cancelled
//   - confirmed  -> cancelled, no_show, consumed
//   - cancelled  -> (terminal, no transiciones)
//   - no_show    -> (terminal, no transiciones)
//   - consumed   -> (terminal, no transiciones)
func CanTransitionReservation(current, next entities.ReservationStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.ReservationStatusPending:
		return next == entities.ReservationStatusConfirmed ||
			next == entities.ReservationStatusCancelled
	case entities.ReservationStatusConfirmed:
		return next == entities.ReservationStatusCancelled ||
			next == entities.ReservationStatusNoShow ||
			next == entities.ReservationStatusConsumed
	default:
		// cancelled, no_show, consumed son terminales.
		return false
	}
}

// spaceCodeRegex valida el formato del codigo de espacio de parqueadero.
// Acepta alfanumericos, guiones y guiones bajos; 1-20 caracteres.
var spaceCodeRegex = regexp.MustCompile(`^[A-Za-z0-9\-_]{1,20}$`)

// ValidateSpaceCode valida que el codigo de un espacio de parqueadero
// sea no vacio y cumpla el formato esperado.
func ValidateSpaceCode(code string) error {
	if code == "" {
		return errors.New("space code is required")
	}
	if !spaceCodeRegex.MatchString(code) {
		return fmt.Errorf("space code %q is invalid (must be 1-20 alphanumeric, hyphens or underscores)", code)
	}
	return nil
}

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23).
//
// Es un duplicado intencional de la funcion de otros modulos: cada
// modulo posee sus propias policies para no acoplar dominios.
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
