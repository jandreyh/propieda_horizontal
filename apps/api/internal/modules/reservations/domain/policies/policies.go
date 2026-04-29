// Package policies contiene funciones puras de logica de negocio del
// modulo reservations.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/saas-ph/api/internal/modules/reservations/domain/entities"
)

// CanTransitionReservation indica si una transicion de status de reserva
// es legal.
//
// Transiciones permitidas:
//   - pending   -> confirmed, cancelled, rejected
//   - confirmed -> cancelled, consumed, no_show
//   - cancelled -> (terminal)
//   - consumed  -> (terminal)
//   - no_show   -> (terminal)
//   - rejected  -> (terminal)
//   - archived  -> (terminal)
func CanTransitionReservation(current, next entities.ReservationStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.ReservationStatusPending:
		return next == entities.ReservationStatusConfirmed ||
			next == entities.ReservationStatusCancelled ||
			next == entities.ReservationStatusRejected
	case entities.ReservationStatusConfirmed:
		return next == entities.ReservationStatusCancelled ||
			next == entities.ReservationStatusConsumed ||
			next == entities.ReservationStatusNoShow
	default:
		// cancelled, consumed, no_show, rejected, archived son terminales.
		return false
	}
}

// ValidateSlotDuration valida que la duracion de un slot no exceda el
// maximo y que end sea posterior a start.
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

// ValidateAttendeesCount valida que el numero de asistentes no exceda
// la capacidad maxima de la zona comun. Si maxCapacity es nil, no hay
// limite.
func ValidateAttendeesCount(attendees *int32, maxCapacity *int32) error {
	if attendees == nil {
		return nil
	}
	if *attendees <= 0 {
		return errors.New("attendees count must be positive")
	}
	if maxCapacity != nil && *attendees > *maxCapacity {
		return fmt.Errorf("attendees count %d exceeds max capacity %d", *attendees, *maxCapacity)
	}
	return nil
}

// ValidateBlackoutWindow valida que un bloqueo tenga fechas coherentes.
func ValidateBlackoutWindow(fromAt, toAt time.Time) error {
	if !toAt.After(fromAt) {
		return errors.New("blackout to_at must be after from_at")
	}
	return nil
}

// IsSlotInBlackout indica si un slot dado cae dentro de un bloqueo.
func IsSlotInBlackout(slotStart, slotEnd, blackoutFrom, blackoutTo time.Time) bool {
	// Overlap: slotStart < blackoutTo AND slotEnd > blackoutFrom
	return slotStart.Before(blackoutTo) && slotEnd.After(blackoutFrom)
}

// GenerateQRCodeHash genera un hash SHA-256 determinista para el QR de
// checkin de una reserva.
//
// Input: reservationID + "|" + commonAreaID + "|" + slotStartAt(RFC3339).
func GenerateQRCodeHash(reservationID, commonAreaID string, slotStartAt time.Time) string {
	input := reservationID + "|" + commonAreaID + "|" + slotStartAt.Format(time.RFC3339)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
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

// CanCreateReservation valida que una zona comun permita crear una
// reserva.
func CanCreateReservation(area entities.CommonArea) error {
	if !area.IsAvailable() {
		return errors.New("common area is not available")
	}
	return nil
}
