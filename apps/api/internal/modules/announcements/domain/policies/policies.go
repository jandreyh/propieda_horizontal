// Package policies contiene funciones puras de logica de negocio del
// modulo announcements: validacion de audiencias y comprobacion de
// expiracion.
//
// Reglas:
//   - Sin dependencias en infra ni HTTP.
//   - Funciones deterministas y testeables sin DB.
package policies

import (
	"errors"
	"fmt"
	"time"

	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
)

// IsExpired indica si un anuncio con expiresAt esta expirado relativo a
// now. Si expiresAt es nil, el anuncio NO expira y la funcion devuelve
// false.
func IsExpired(now time.Time, expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return !expiresAt.After(now)
}

// UserScopes agrupa los scopes que un usuario tiene asignados (sus
// roles, las structures bajo su acceso y las unidades de las que es
// residente/owner).
type UserScopes struct {
	RoleIDs      []string
	StructureIDs []string
	UnitIDs      []string
}

// MatchesAudience indica si UserScopes contiene al menos un scope que
// matche con alguna audiencia provista. La audiencia 'global' siempre
// matchea.
func MatchesAudience(audiences []entities.Audience, scopes UserScopes) bool {
	for _, a := range audiences {
		switch a.TargetType {
		case entities.TargetGlobal:
			return true
		case entities.TargetRole:
			if a.TargetID != nil && contains(scopes.RoleIDs, *a.TargetID) {
				return true
			}
		case entities.TargetStructure:
			if a.TargetID != nil && contains(scopes.StructureIDs, *a.TargetID) {
				return true
			}
		case entities.TargetUnit:
			if a.TargetID != nil && contains(scopes.UnitIDs, *a.TargetID) {
				return true
			}
		}
	}
	return false
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

// ValidateAudienceCoherence verifica que (target_type, target_id) sea
// coherente:
//   - target_type='global' -> target_id == nil.
//   - cualquier otro       -> target_id != nil y no vacio.
func ValidateAudienceCoherence(targetType entities.TargetType, targetID *string) error {
	if !targetType.IsValid() {
		return fmt.Errorf("invalid target_type %q", targetType)
	}
	if targetType == entities.TargetGlobal {
		if targetID != nil && *targetID != "" {
			return errors.New("target_id must be null when target_type is 'global'")
		}
		return nil
	}
	if targetID == nil || *targetID == "" {
		return fmt.Errorf("target_id is required when target_type is %q", targetType)
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
