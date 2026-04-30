// Package domain define las interfaces de tenant_members.
package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/tenant_members/domain/entities"
)

// ErrLinkNotFound se emite cuando no existe un tenant_user_link para el id.
var ErrLinkNotFound = errors.New("tenant_members: link not found")

// ErrAlreadyLinked se emite cuando ya existe un link para ese platform_user_id.
var ErrAlreadyLinked = errors.New("tenant_members: user already linked")

// LinkRepository abstrae el acceso a `tenant_user_links` (DB del tenant).
type LinkRepository interface {
	// Create inserta un nuevo link. Devuelve ErrAlreadyLinked si la
	// constraint UNIQUE(platform_user_id) choca.
	Create(ctx context.Context, in CreateLink) (*entities.TenantMember, error)
	// List devuelve todos los links activos del tenant (no-soft-deleted).
	List(ctx context.Context) ([]entities.TenantMember, error)
	// FindByID lee el link por su id.
	FindByID(ctx context.Context, id uuid.UUID) (*entities.TenantMember, error)
	// Update modifica role/unit/status. Usa optimistic locking (version).
	Update(ctx context.Context, in UpdateLink) (*entities.TenantMember, error)
	// Block fija status='blocked'.
	Block(ctx context.Context, id uuid.UUID) error
}

// CreateLink son los campos para insertar un nuevo link.
type CreateLink struct {
	PlatformUserID uuid.UUID
	Role           string
	PrimaryUnitID  *uuid.UUID
}

// UpdateLink son los campos modificables de un link existente.
type UpdateLink struct {
	ID            uuid.UUID
	Role          string
	PrimaryUnitID *uuid.UUID
	Version       int32 // version esperada (optimistic lock)
}

// EnricherRepository abstrae la consulta a la DB central para hidratar
// los datos del PlatformUser referenciado por un link.
type EnricherRepository interface {
	// Hydrate completa Names/LastNames/Email/PublicCode de los miembros
	// dado su platform_user_id. Hace UN solo query batch.
	Hydrate(ctx context.Context, members []entities.TenantMember) ([]entities.TenantMember, error)
	// FindPlatformUserIDByCode resuelve un public_code al user id en
	// la DB central. Devuelve domain.ErrPlatformUserNotFound si no existe.
	FindPlatformUserIDByCode(ctx context.Context, code string) (uuid.UUID, string, string, string, error)
}

// ErrPlatformUserNotFound se emite cuando el public_code no resuelve.
var ErrPlatformUserNotFound = errors.New("tenant_members: platform user not found")
