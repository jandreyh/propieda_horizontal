// Package policies agrupa funciones puras que codifican reglas de
// negocio del modulo identity. No tienen efectos ni dependen de IO; son
// directamente testeables sin DB ni HTTP.
package policies

import (
	"time"

	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
)

// MaxFailedAttempts es el numero de intentos fallidos consecutivos
// despues del cual el usuario queda bloqueado temporalmente.
const MaxFailedAttempts = 5

// LockoutDuration es el tiempo que dura el bloqueo automatico tras
// alcanzar MaxFailedAttempts.
const LockoutDuration = 15 * time.Minute

// IsLocked devuelve true cuando el usuario tiene un locked_until vigente
// respecto a la referencia now.
//
// La comparacion es estricta (locked_until > now) para que un usuario
// cuyo lockout justo expira pueda iniciar sesion en ese mismo instante.
func IsLocked(u *entities.User, now time.Time) bool {
	if u == nil || u.LockedUntil == nil {
		return false
	}
	return u.LockedUntil.After(now)
}

// CanLogin devuelve true cuando el usuario esta en condiciones logicas
// de iniciar sesion: existe, no esta soft-deleted, status active, y no
// tiene lockout vigente.
func CanLogin(u *entities.User, now time.Time) bool {
	if u == nil {
		return false
	}
	if !u.IsActive() {
		return false
	}
	if IsLocked(u, now) {
		return false
	}
	return true
}

// WouldExceedFailedAttempts devuelve true si sumar uno al contador
// actual alcanza o supera MaxFailedAttempts. Se invoca antes de
// incrementar para decidir si debe iniciarse el lockout.
func WouldExceedFailedAttempts(currentAttempts int) bool {
	return currentAttempts+1 >= MaxFailedAttempts
}

// NextLockUntil calcula el timestamp hasta el que el usuario debe quedar
// bloqueado partiendo del momento now y la duracion estandar.
func NextLockUntil(now time.Time) time.Time {
	return now.Add(LockoutDuration)
}

// ShouldRequireMFA devuelve true si el usuario debe pasar por la
// segunda etapa de autenticacion antes de obtener tokens de sesion.
func ShouldRequireMFA(u *entities.User) bool {
	return u != nil && u.HasMFA()
}
