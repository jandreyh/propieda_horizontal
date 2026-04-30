package entities

import (
	"time"

	"github.com/google/uuid"
)

// PlatformSession representa una fila de `platform_user_sessions`. La
// fila se identifica con `id` (uuid) y guarda el `token_hash` SHA-256 del
// refresh token plano. El access token (JWT) NO se persiste; lo unico
// que se persiste es el handle del refresh para soportar revocation y
// rotacion.
type PlatformSession struct {
	ID               uuid.UUID
	PlatformUserID   uuid.UUID
	TokenHash        string
	ParentSessionID  *uuid.UUID
	UserAgent        *string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	RevocationReason *string
	Status           string
}

// IsActive es true si la sesion sigue valida (no expirada ni revocada).
func (s PlatformSession) IsActive(now time.Time) bool {
	if s.Status != "active" {
		return false
	}
	if s.RevokedAt != nil {
		return false
	}
	return now.Before(s.ExpiresAt)
}
