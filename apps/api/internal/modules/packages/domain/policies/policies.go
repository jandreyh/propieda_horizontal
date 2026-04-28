// Package policies contiene funciones puras de logica de negocio del
// modulo packages.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"

	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
)

// RequiresEvidence indica si la categoria del paquete obliga a evidencia
// visual (foto) en el momento de recepcion. Hoy: solo "Refrigerado".
//
// Si la categoria es nil (paquete sin categorizar), no se exige
// evidencia.
func RequiresEvidence(c *entities.PackageCategory) bool {
	if c == nil {
		return false
	}
	return c.RequiresEvidence
}

// CanTransition indica si una transicion de status es legal.
//
// Reglas:
//   - received -> delivered : permitido (entrega exitosa).
//   - received -> returned  : permitido (devolucion al transportador).
//   - cualquier transicion desde delivered o returned : NO permitida.
//   - mismo status : NO permitido (no idempotente a nivel de transicion;
//     idempotency hit lo manejan los usecases con cache de respuestas).
func CanTransition(current, next entities.PackageStatus) bool {
	if current == next {
		return false
	}
	if current != entities.PackageStatusReceived {
		return false
	}
	return next == entities.PackageStatusDelivered ||
		next == entities.PackageStatusReturned
}

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23).
//
// Es un duplicado intencional de la funcion de access_control: cada
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

// ValidateRecipientName valida que el nombre del destinatario sea no
// vacio y razonable.
func ValidateRecipientName(s string) error {
	if s == "" {
		return errors.New("recipient_name is required")
	}
	if len(s) > 200 {
		return fmt.Errorf("recipient_name too long (max 200, got %d)", len(s))
	}
	return nil
}
