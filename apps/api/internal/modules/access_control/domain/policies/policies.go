// Package policies contiene funciones puras de logica de negocio del
// modulo access_control.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"time"

	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
)

// IsExpired indica si un instante `t` ya quedo en el pasado respecto a
// `now`. Util para chequear `expires_at` de pre-registros y blacklist.
//
// Si t es el zero-value (time.Time{}), se considera no-expirado.
func IsExpired(t time.Time, now time.Time) bool {
	if t.IsZero() {
		return false
	}
	return !t.After(now)
}

// RequiresPhoto devuelve true si el origen del checkin obliga a foto del
// documento. Hoy: solo el flujo manual.
func RequiresPhoto(source string) bool {
	return source == string(entities.VisitorEntrySourceManual)
}

// BlacklistMatch indica si una entrada de blacklist sigue vigente en
// `now`. Considera vigente si:
//   - no esta archivada,
//   - no esta soft-deleted,
//   - no expiro (o expires_at es nil).
func BlacklistMatch(b entities.BlacklistEntry, now time.Time) bool {
	if b.IsArchived() {
		return false
	}
	if b.ExpiresAt != nil && IsExpired(*b.ExpiresAt, now) {
		return false
	}
	return true
}

// ValidateDocumentType verifica que el tipo de documento este en el set
// permitido por el modulo (alineado con identity).
func ValidateDocumentType(t string) error {
	if t == "" {
		return errors.New("document_type is required")
	}
	if !entities.DocumentType(t).IsValid() {
		return fmt.Errorf("invalid document_type %q (allowed: CC, CE, PA, TI, RC, NIT)", t)
	}
	return nil
}

// ValidateDocumentNumber verifica que un numero de documento sea no
// vacio y razonable en longitud.
func ValidateDocumentNumber(n string) error {
	if n == "" {
		return errors.New("document_number is required")
	}
	if len(n) > 32 {
		return fmt.Errorf("document_number too long (max 32, got %d)", len(n))
	}
	return nil
}

// ValidateUUID hace una validacion sintactica minima del formato UUID
// (36 caracteres con guiones en posiciones 8/13/18/23).
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

// ValidateMaxUses acepta nil (default=1) o un entero >=1.
func ValidateMaxUses(maxUses *int32) (int32, error) {
	if maxUses == nil {
		return 1, nil
	}
	if *maxUses < 1 {
		return 0, fmt.Errorf("max_uses must be >= 1 (got %d)", *maxUses)
	}
	return *maxUses, nil
}
