// Package entities define las entidades de dominio del modulo
// platform_identity. Sin tags JSON ni de DB.
package entities

import (
	"time"

	"github.com/google/uuid"
)

// PlatformUser es la identidad global de una persona en plataforma.
// Vive en la DB central (`platform_users`) y se reusa entre N conjuntos.
type PlatformUser struct {
	ID                  uuid.UUID
	DocumentType        string
	DocumentNumber      string
	Names               string
	LastNames           string
	Email               string
	Phone               *string
	PhotoURL            *string
	PasswordHash        string
	MFASecret           *string
	MFAEnrolledAt       *time.Time
	PublicCode          string
	FailedLoginAttempts int32
	LockedUntil         *time.Time
	LastLoginAt         *time.Time
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Membership representa la pertenencia de un PlatformUser a un tenant.
// Es el dato que alimenta el selector y el JWT.
type Membership struct {
	TenantID     uuid.UUID
	TenantSlug   string
	TenantName   string
	LogoURL      *string
	PrimaryColor *string
	Role         string
	Status       string
}

// IsActive indica si la identidad puede autenticarse.
func (u PlatformUser) IsActive() bool {
	return u.Status == "active"
}

// IsLocked indica si la cuenta esta temporalmente bloqueada por intentos fallidos.
func (u PlatformUser) IsLocked(now time.Time) bool {
	if u.LockedUntil == nil {
		return false
	}
	return now.Before(*u.LockedUntil)
}
