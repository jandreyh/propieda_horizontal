// Package domain define las interfaces que la capa de aplicacion del
// modulo platform_identity exige a la capa de infraestructura.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
)

// ErrUserNotFound es la respuesta canonica del repo cuando no existe el
// PlatformUser solicitado.
var ErrUserNotFound = errors.New("platform_identity: user not found")

// PlatformUserRepository abstrae el acceso a la tabla central
// `platform_users` (DB Control Plane).
type PlatformUserRepository interface {
	// FindByEmail busca por email case-insensitive.
	FindByEmail(ctx context.Context, email string) (*entities.PlatformUser, error)
	// FindByDocument busca por par documento.
	FindByDocument(ctx context.Context, docType, docNumber string) (*entities.PlatformUser, error)
	// FindByID busca por UUID interno.
	FindByID(ctx context.Context, id uuid.UUID) (*entities.PlatformUser, error)
	// FindByPublicCode busca por codigo unico de vinculacion a conjuntos.
	FindByPublicCode(ctx context.Context, code string) (*entities.PlatformUser, error)
	// MarkLoginSuccess sella last_login_at = when y resetea el contador
	// de intentos fallidos.
	MarkLoginSuccess(ctx context.Context, id uuid.UUID, when time.Time) error
	// IncrementFailedLogin suma 1 al contador y bloquea hasta locked_until
	// si supera el umbral. Devuelve el nuevo contador y el lock-until si
	// existe.
	IncrementFailedLogin(ctx context.Context, id uuid.UUID) (int32, *time.Time, error)
	// ListMemberships devuelve las membresias activas del usuario,
	// alimentando el selector y el JWT.
	ListMemberships(ctx context.Context, userID uuid.UUID) ([]entities.Membership, error)
	// HasMembership verifica si el usuario tiene acceso activo al tenant
	// identificado por slug.
	HasMembership(ctx context.Context, userID uuid.UUID, slug string) (bool, error)
}

// PushDeviceRepository abstrae el acceso a `platform_push_devices` para
// el manejo de tokens FCM/APNs/Web a nivel plataforma.
type PushDeviceRepository interface {
	// Register hace upsert (token unique por usuario): si existe, marca
	// last_seen_at=now y limpia revoked_at; si no, inserta.
	Register(ctx context.Context, userID uuid.UUID, token, platform string, label *string) (*entities.PushDevice, error)
	// Revoke marca un device revocado para no enviarle mas notifs.
	Revoke(ctx context.Context, deviceID, userID uuid.UUID) error
	// List devuelve los devices activos de un usuario.
	List(ctx context.Context, userID uuid.UUID) ([]entities.PushDevice, error)
}
