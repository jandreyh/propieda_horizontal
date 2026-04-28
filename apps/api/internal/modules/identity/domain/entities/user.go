// Package entities contiene las entidades puras del dominio identity.
//
// Estas estructuras NO llevan tags JSON ni DB. Se mapean explicitamente a
// DTOs (capa application) y a structs de sqlc (capa infrastructure).
package entities

import "time"

// DocumentType representa el tipo de documento de identidad aceptado por
// el sistema colombiano de propiedad horizontal.
type DocumentType string

// Tipos de documento permitidos. Mantener sincronizado con el CHECK
// constraint de la tabla users.
const (
	DocumentTypeCC  DocumentType = "CC"  // Cedula de ciudadania
	DocumentTypeCE  DocumentType = "CE"  // Cedula de extranjeria
	DocumentTypePA  DocumentType = "PA"  // Pasaporte
	DocumentTypeTI  DocumentType = "TI"  // Tarjeta de identidad
	DocumentTypeRC  DocumentType = "RC"  // Registro civil
	DocumentTypeNIT DocumentType = "NIT" // NIT (juridica)
)

// IsValid devuelve true si dt es uno de los valores permitidos.
func (dt DocumentType) IsValid() bool {
	switch dt {
	case DocumentTypeCC, DocumentTypeCE, DocumentTypePA,
		DocumentTypeTI, DocumentTypeRC, DocumentTypeNIT:
		return true
	}
	return false
}

// UserStatus controla la disponibilidad operativa del usuario.
type UserStatus string

// Estados permitidos. Sincronizar con CHECK constraint.
const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

// User representa al actor autenticable del tenant. Es la entidad raiz
// del agregado identity. NO contiene secretos en logs (los marshallers
// de DTO se encargan de ocultar password_hash y mfa_secret).
type User struct {
	ID                  string
	DocumentType        DocumentType
	DocumentNumber      string
	Names               string
	LastNames           string
	Email               *string
	Phone               *string
	PasswordHash        string
	MFASecret           *string
	MFAEnrolledAt       *time.Time
	FailedLoginAttempts int
	LockedUntil         *time.Time
	LastLoginAt         *time.Time
	Status              UserStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	CreatedBy           *string
	UpdatedBy           *string
	DeletedBy           *string
	Version             int
}

// HasMFA indica si el usuario ya tiene MFA enrolado (mfa_secret no
// vacio). El usecase de login utiliza este flag para decidir si emite un
// pre_auth_token o si entrega tokens de sesion completa.
func (u *User) HasMFA() bool {
	if u == nil || u.MFASecret == nil {
		return false
	}
	return *u.MFASecret != ""
}

// IsActive devuelve true cuando el usuario puede iniciar sesion segun
// los flags persistidos (no incluye lockout temporal por intentos
// fallidos — eso lo cubre IsLocked).
func (u *User) IsActive() bool {
	if u == nil {
		return false
	}
	if u.DeletedAt != nil {
		return false
	}
	return u.Status == UserStatusActive
}
