package usecases

import (
	"context"
	"errors"

	apperrors "github.com/saas-ph/api/internal/platform/errors"

	"github.com/saas-ph/api/internal/modules/tenant_config/domain"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/policies"
)

// GetBranding devuelve la fila singleton.
type GetBranding struct {
	Repo domain.BrandingRepository
}

// Execute delega al repo y mapea errores conocidos.
func (u GetBranding) Execute(ctx context.Context) (entities.Branding, error) {
	b, err := u.Repo.Get(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrBrandingNotFound) {
			return entities.Branding{}, apperrors.NotFound("branding not found")
		}
		return entities.Branding{}, apperrors.Internal("failed to load branding")
	}
	return b, nil
}

// UpdateBranding actualiza la fila singleton (concurrencia optimista).
type UpdateBranding struct {
	Repo domain.BrandingRepository
}

// UpdateBrandingInput es el input del usecase.
type UpdateBrandingInput struct {
	DisplayName     string
	LogoURL         *string
	PrimaryColor    *string
	SecondaryColor  *string
	Timezone        string
	Locale          string
	ActorID         string
	ExpectedVersion int32
}

// Execute valida y delega.
func (u UpdateBranding) Execute(ctx context.Context, in UpdateBrandingInput) (entities.Branding, error) {
	if err := policies.ValidateDisplayName(in.DisplayName); err != nil {
		return entities.Branding{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateHexColor(in.PrimaryColor); err != nil {
		return entities.Branding{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateHexColor(in.SecondaryColor); err != nil {
		return entities.Branding{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateTimezone(in.Timezone); err != nil {
		return entities.Branding{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateLocale(in.Locale); err != nil {
		return entities.Branding{}, apperrors.BadRequest(err.Error())
	}
	if in.ExpectedVersion <= 0 {
		return entities.Branding{}, apperrors.BadRequest("expected_version must be > 0")
	}

	b, err := u.Repo.Update(ctx, domain.UpdateBrandingInput{
		DisplayName:     in.DisplayName,
		LogoURL:         in.LogoURL,
		PrimaryColor:    in.PrimaryColor,
		SecondaryColor:  in.SecondaryColor,
		Timezone:        in.Timezone,
		Locale:          in.Locale,
		ActorID:         in.ActorID,
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBrandingNotFound):
			return entities.Branding{}, apperrors.NotFound("branding not found")
		case errors.Is(err, domain.ErrVersionMismatch):
			return entities.Branding{}, apperrors.Conflict("branding version mismatch")
		default:
			return entities.Branding{}, apperrors.Internal("failed to update branding")
		}
	}
	return b, nil
}
