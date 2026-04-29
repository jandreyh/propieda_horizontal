// Package policies contiene funciones puras de logica de negocio del
// modulo pqrs.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"

	"github.com/saas-ph/api/internal/modules/pqrs/domain/entities"
)

// CanTransitionTicket indica si una transicion de status de ticket
// es legal.
//
// Transiciones permitidas:
//   - radicado    -> en_estudio, escalado, cancelado
//   - en_estudio  -> respondido, escalado, cancelado
//   - respondido  -> cerrado
//   - cerrado     -> (terminal, no transiciones)
//   - escalado    -> en_estudio, cancelado
//   - cancelado   -> (terminal, no transiciones)
func CanTransitionTicket(current, next entities.TicketStatus) bool {
	if current == next {
		return false
	}
	switch current {
	case entities.TicketStatusRadicado:
		return next == entities.TicketStatusEnEstudio ||
			next == entities.TicketStatusEscalado ||
			next == entities.TicketStatusCancelado
	case entities.TicketStatusEnEstudio:
		return next == entities.TicketStatusRespondido ||
			next == entities.TicketStatusEscalado ||
			next == entities.TicketStatusCancelado
	case entities.TicketStatusRespondido:
		return next == entities.TicketStatusCerrado
	case entities.TicketStatusEscalado:
		return next == entities.TicketStatusEnEstudio ||
			next == entities.TicketStatusCancelado
	default:
		// cerrado, cancelado son terminales.
		return false
	}
}

// ValidateRating valida que el rating este entre 1 y 5.
func ValidateRating(rating *int32) error {
	if rating == nil {
		return nil
	}
	if *rating < 1 || *rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	return nil
}

// ValidateAnonymous rechaza tickets anonimos en V1.
func ValidateAnonymous(isAnonymous bool) error {
	if isAnonymous {
		return errors.New("anonymous tickets are not supported in V1")
	}
	return nil
}

// ValidateCloseBeforeResponse rechaza el cierre de un ticket sin
// respuesta oficial previa.
func ValidateCloseBeforeResponse(ticket entities.Ticket, hasOfficialResponse bool) error {
	if !hasOfficialResponse {
		return errors.New("cannot close ticket without an official response")
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
