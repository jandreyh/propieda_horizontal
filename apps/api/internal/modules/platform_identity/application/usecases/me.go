package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
)

// ErrUserMismatch lo emite Me cuando el subject del JWT no resuelve a un
// PlatformUser activo (por ejemplo, suspendido despues de emitir el JWT).
var ErrUserMismatch = errors.New("platform_identity: user no longer accessible")

// MeDeps son las dependencias de MeUseCase.
type MeDeps struct {
	Users domain.PlatformUserRepository
}

// MeUseCase implementa GET /me — devuelve los datos globales de la
// persona autenticada.
type MeUseCase struct {
	deps MeDeps
}

// NewMeUseCase construye el usecase.
func NewMeUseCase(deps MeDeps) *MeUseCase {
	return &MeUseCase{deps: deps}
}

// Execute resuelve el subject del JWT a un MeResponse.
func (uc *MeUseCase) Execute(ctx context.Context, subject string) (dto.MeResponse, error) {
	id, err := uuid.Parse(subject)
	if err != nil {
		return dto.MeResponse{}, ErrInvalidInput
	}
	user, err := uc.deps.Users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.MeResponse{}, ErrUserMismatch
		}
		return dto.MeResponse{}, fmt.Errorf("%w: find by id: %w", ErrInternal, err)
	}
	if !user.IsActive() {
		return dto.MeResponse{}, ErrAccountInactive
	}
	return dto.MeResponse{
		ID:             user.ID.String(),
		DocumentType:   user.DocumentType,
		DocumentNumber: user.DocumentNumber,
		Names:          user.Names,
		LastNames:      user.LastNames,
		Email:          user.Email,
		Phone:          user.Phone,
		PhotoURL:       user.PhotoURL,
		PublicCode:     user.PublicCode,
		MFAEnrolledAt:  user.MFAEnrolledAt,
		LastLoginAt:    user.LastLoginAt,
	}, nil
}

// ListMembershipsDeps son las dependencias de ListMembershipsUseCase.
type ListMembershipsDeps struct {
	Users domain.PlatformUserRepository
}

// ListMembershipsUseCase implementa GET /me/memberships.
type ListMembershipsUseCase struct {
	deps ListMembershipsDeps
}

// NewListMembershipsUseCase construye el usecase.
func NewListMembershipsUseCase(deps ListMembershipsDeps) *ListMembershipsUseCase {
	return &ListMembershipsUseCase{deps: deps}
}

// Execute devuelve las membresias activas de la persona.
func (uc *ListMembershipsUseCase) Execute(ctx context.Context, subject string) (dto.MembershipsResponse, error) {
	id, err := uuid.Parse(subject)
	if err != nil {
		return dto.MembershipsResponse{}, ErrInvalidInput
	}
	rows, err := uc.deps.Users.ListMemberships(ctx, id)
	if err != nil {
		return dto.MembershipsResponse{}, fmt.Errorf("%w: list memberships: %w", ErrInternal, err)
	}
	items := make([]dto.MembershipDTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, dto.MembershipDTO{
			TenantID:     m.TenantID.String(),
			TenantSlug:   m.TenantSlug,
			TenantName:   m.TenantName,
			LogoURL:      m.LogoURL,
			PrimaryColor: m.PrimaryColor,
			Role:         m.Role,
			Status:       m.Status,
		})
	}
	return dto.MembershipsResponse{Items: items}, nil
}
