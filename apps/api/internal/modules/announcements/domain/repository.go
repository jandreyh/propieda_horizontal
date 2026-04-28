// Package domain define las interfaces (puertos) del modulo
// announcements que la capa de aplicacion consume y que infra implementa.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
)

// ErrAnnouncementNotFound se devuelve cuando no existe (o esta archivado)
// un anuncio por el id consultado.
var ErrAnnouncementNotFound = errors.New("announcements: announcement not found")

// ErrInvalidAudience se devuelve cuando una audiencia no es coherente
// (target_type=global con target_id no nulo, o target_type<>global con
// target_id nulo).
var ErrInvalidAudience = errors.New("announcements: invalid audience")

// ErrTitleRequired se devuelve cuando el title viene vacio.
var ErrTitleRequired = errors.New("announcements: title is required")

// ErrBodyRequired se devuelve cuando el body viene vacio.
var ErrBodyRequired = errors.New("announcements: body is required")

// CreateAnnouncementInput agrupa los datos necesarios para persistir un
// anuncio nuevo. La validacion la hace la capa de aplicacion.
type CreateAnnouncementInput struct {
	Title             string
	Body              string
	PublishedByUserID string
	Pinned            bool
	ExpiresAt         *time.Time
}

// AddAudienceInput agrupa los datos para insertar una audiencia de
// anuncio.
type AddAudienceInput struct {
	AnnouncementID string
	TargetType     entities.TargetType
	TargetID       *string
	ActorID        string
}

// FeedQuery encapsula los parametros para listar el feed visible para un
// usuario.
type FeedQuery struct {
	UserID       string
	RoleIDs      []string
	StructureIDs []string
	UnitIDs      []string
	Limit        int32
	Offset       int32
}

// AnnouncementRepository es el puerto que persiste anuncios.
type AnnouncementRepository interface {
	// Create inserta un anuncio. Esta llamada se espera dentro de una
	// transaccion (junto con AddAudience) cuando el caller ya abrio una.
	Create(ctx context.Context, in CreateAnnouncementInput) (entities.Announcement, error)
	// GetByID devuelve un anuncio activo por id, o ErrAnnouncementNotFound.
	GetByID(ctx context.Context, id string) (entities.Announcement, error)
	// Archive marca el anuncio como soft-deleted (status='archived').
	Archive(ctx context.Context, id, actorID string) (entities.Announcement, error)
	// ListFeedForUser devuelve los anuncios visibles para el usuario
	// dados sus scopes (role/structure/unit). Filtra publicados, no
	// expirados y con al menos una audiencia que matche.
	ListFeedForUser(ctx context.Context, q FeedQuery) ([]entities.Announcement, error)
}

// AudienceRepository es el puerto que persiste audiencias de anuncios.
type AudienceRepository interface {
	// Add inserta una audiencia para un anuncio.
	Add(ctx context.Context, in AddAudienceInput) (entities.Audience, error)
	// ListByAnnouncement devuelve las audiencias activas de un anuncio.
	ListByAnnouncement(ctx context.Context, announcementID string) ([]entities.Audience, error)
}

// AckRepository es el puerto que persiste confirmaciones de lectura.
type AckRepository interface {
	// Acknowledge inserta (idempotente) una confirmacion de lectura para
	// (announcementID, userID).
	Acknowledge(ctx context.Context, announcementID, userID string) (entities.Ack, error)
}
