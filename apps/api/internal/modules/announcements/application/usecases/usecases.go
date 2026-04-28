// Package usecases orquesta la logica de aplicacion del modulo
// announcements. Cada usecase recibe sus dependencias por inyeccion
// (interfaces) y NO conoce HTTP ni la base.
package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/announcements/domain"
	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
	"github.com/saas-ph/api/internal/modules/announcements/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// TxRunner abre una transaccion (Tenant DB) y ejecuta fn dentro de ella.
// La implementacion concreta vive en infrastructure/persistence; el
// usecase la consume via interface para no acoplarse a pgx.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// CreateAnnouncementInput es el input del usecase (sin tags JSON).
type CreateAnnouncementInput struct {
	Title             string
	Body              string
	PublishedByUserID string
	Pinned            bool
	ExpiresAt         *time.Time
	Audiences         []AudienceInput
}

// AudienceInput describe una audiencia a crear con el anuncio.
type AudienceInput struct {
	TargetType string
	TargetID   *string
}

// CreateAnnouncementOutput es la respuesta del usecase con el anuncio
// recien creado y sus audiencias.
type CreateAnnouncementOutput struct {
	Announcement entities.Announcement
	Audiences    []entities.Audience
}

// CreateAnnouncement orquesta la creacion atomica de un anuncio + sus
// audiencias dentro de una transaccion del Tenant DB.
type CreateAnnouncement struct {
	Announcements domain.AnnouncementRepository
	Audiences     domain.AudienceRepository
	TxRunner      TxRunner
}

// Execute valida el input y persiste el anuncio + audiencias.
func (u CreateAnnouncement) Execute(ctx context.Context, in CreateAnnouncementInput) (CreateAnnouncementOutput, error) {
	if strings.TrimSpace(in.Title) == "" {
		return CreateAnnouncementOutput{}, apperrors.BadRequest(domain.ErrTitleRequired.Error())
	}
	if strings.TrimSpace(in.Body) == "" {
		return CreateAnnouncementOutput{}, apperrors.BadRequest(domain.ErrBodyRequired.Error())
	}
	if err := policies.ValidateUUID(in.PublishedByUserID); err != nil {
		return CreateAnnouncementOutput{}, apperrors.BadRequest("published_by_user_id: " + err.Error())
	}
	if len(in.Audiences) == 0 {
		return CreateAnnouncementOutput{}, apperrors.BadRequest(domain.ErrInvalidAudience.Error() + ": at least one audience required")
	}
	for i, a := range in.Audiences {
		tt := entities.TargetType(a.TargetType)
		if err := policies.ValidateAudienceCoherence(tt, a.TargetID); err != nil {
			return CreateAnnouncementOutput{}, apperrors.BadRequest(
				"audiences[" + itoa(i) + "]: " + err.Error())
		}
	}

	out := CreateAnnouncementOutput{}
	op := func(ctx context.Context) error {
		ann, err := u.Announcements.Create(ctx, domain.CreateAnnouncementInput{
			Title:             strings.TrimSpace(in.Title),
			Body:              strings.TrimSpace(in.Body),
			PublishedByUserID: in.PublishedByUserID,
			Pinned:            in.Pinned,
			ExpiresAt:         in.ExpiresAt,
		})
		if err != nil {
			return err
		}
		auds := make([]entities.Audience, 0, len(in.Audiences))
		for _, a := range in.Audiences {
			ad, err := u.Audiences.Add(ctx, domain.AddAudienceInput{
				AnnouncementID: ann.ID,
				TargetType:     entities.TargetType(a.TargetType),
				TargetID:       a.TargetID,
				ActorID:        in.PublishedByUserID,
			})
			if err != nil {
				return err
			}
			auds = append(auds, ad)
		}
		out = CreateAnnouncementOutput{Announcement: ann, Audiences: auds}
		return nil
	}

	// Si hay TxRunner configurado, lo usamos; si no, ejecutamos sin
	// transaccion (los repos resolveran el pool del tenant directamente).
	// En produccion el TxRunner SIEMPRE esta presente; el fallback es
	// util para tests con repos en memoria.
	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, op); err != nil {
			return CreateAnnouncementOutput{}, mapErr(err, "failed to create announcement")
		}
		return out, nil
	}
	if err := op(ctx); err != nil {
		return CreateAnnouncementOutput{}, mapErr(err, "failed to create announcement")
	}
	return out, nil
}

// GetAnnouncement devuelve un anuncio + sus audiencias.
type GetAnnouncement struct {
	Announcements domain.AnnouncementRepository
	Audiences     domain.AudienceRepository
}

// GetAnnouncementOutput agrupa el anuncio y sus audiencias.
type GetAnnouncementOutput struct {
	Announcement entities.Announcement
	Audiences    []entities.Audience
}

// Execute valida el id y delega al repo.
func (u GetAnnouncement) Execute(ctx context.Context, id string) (GetAnnouncementOutput, error) {
	if err := policies.ValidateUUID(id); err != nil {
		return GetAnnouncementOutput{}, apperrors.BadRequest(err.Error())
	}
	ann, err := u.Announcements.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAnnouncementNotFound) {
			return GetAnnouncementOutput{}, apperrors.NotFound("announcement not found")
		}
		return GetAnnouncementOutput{}, apperrors.Internal("failed to load announcement")
	}
	auds, err := u.Audiences.ListByAnnouncement(ctx, id)
	if err != nil {
		return GetAnnouncementOutput{}, apperrors.Internal("failed to load audiences")
	}
	return GetAnnouncementOutput{Announcement: ann, Audiences: auds}, nil
}

// ArchiveAnnouncement marca un anuncio como archivado (soft-delete).
type ArchiveAnnouncement struct {
	Announcements domain.AnnouncementRepository
}

// ArchiveAnnouncementInput es el input del usecase.
type ArchiveAnnouncementInput struct {
	ID      string
	ActorID string
}

// Execute valida y delega al repo.
func (u ArchiveAnnouncement) Execute(ctx context.Context, in ArchiveAnnouncementInput) (entities.Announcement, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.Announcement{}, apperrors.BadRequest(err.Error())
	}
	ann, err := u.Announcements.Archive(ctx, in.ID, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrAnnouncementNotFound) {
			return entities.Announcement{}, apperrors.NotFound("announcement not found")
		}
		return entities.Announcement{}, apperrors.Internal("failed to archive announcement")
	}
	return ann, nil
}

// Acknowledge confirma la lectura de un anuncio por un usuario.
// Idempotente: insertar dos veces NO falla.
type Acknowledge struct {
	Acks domain.AckRepository
}

// AcknowledgeInput es el input del usecase.
type AcknowledgeInput struct {
	AnnouncementID string
	UserID         string
}

// Execute valida y delega al repo.
func (u Acknowledge) Execute(ctx context.Context, in AcknowledgeInput) (entities.Ack, error) {
	if err := policies.ValidateUUID(in.AnnouncementID); err != nil {
		return entities.Ack{}, apperrors.BadRequest("announcement_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UserID); err != nil {
		return entities.Ack{}, apperrors.BadRequest("user_id: " + err.Error())
	}
	ack, err := u.Acks.Acknowledge(ctx, in.AnnouncementID, in.UserID)
	if err != nil {
		return entities.Ack{}, apperrors.Internal("failed to acknowledge announcement")
	}
	return ack, nil
}

// ListFeed devuelve el feed visible para un usuario dado.
type ListFeed struct {
	Announcements domain.AnnouncementRepository
}

// FeedInput es el input del usecase.
type FeedInput struct {
	UserID       string
	RoleIDs      []string
	StructureIDs []string
	UnitIDs      []string
	Limit        int32
	Offset       int32
}

// FeedOutput agrupa el resultado del feed.
type FeedOutput struct {
	Items []entities.Announcement
	Total int
}

// Execute valida defaults y delega al repo.
func (u ListFeed) Execute(ctx context.Context, in FeedInput) (FeedOutput, error) {
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 20
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	items, err := u.Announcements.ListFeedForUser(ctx, domain.FeedQuery{
		UserID:       in.UserID,
		RoleIDs:      in.RoleIDs,
		StructureIDs: in.StructureIDs,
		UnitIDs:      in.UnitIDs,
		Limit:        in.Limit,
		Offset:       in.Offset,
	})
	if err != nil {
		return FeedOutput{}, apperrors.Internal("failed to list feed")
	}
	return FeedOutput{Items: items, Total: len(items)}, nil
}

// --- helpers ---

// mapErr traduce errores conocidos del dominio a apperrors. Si el error
// ya es un apperrors.Problem, lo respeta tal cual.
func mapErr(err error, fallback string) error {
	var p apperrors.Problem
	if errors.As(err, &p) {
		return p
	}
	switch {
	case errors.Is(err, domain.ErrAnnouncementNotFound):
		return apperrors.NotFound("announcement not found")
	case errors.Is(err, domain.ErrInvalidAudience):
		return apperrors.BadRequest(err.Error())
	case errors.Is(err, tenantctx.ErrNoTenant):
		return apperrors.Internal("tenant not resolved")
	case errors.Is(err, pgx.ErrNoRows):
		return apperrors.NotFound("not found")
	}
	return apperrors.Internal(fallback)
}

// itoa convierte un int pequeno a string sin importar strconv (mantener
// el paquete liviano).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
