package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// ErrMembershipMissing lo emite SwitchTenant cuando la persona no tiene
// membership activa en el tenant slug solicitado.
var ErrMembershipMissing = errors.New("platform_identity: membership missing for tenant")

// SwitchTenantDeps son las dependencias de SwitchTenantUseCase.
type SwitchTenantDeps struct {
	Users  domain.PlatformUserRepository
	Signer *jwtsign.Signer
	Now    func() time.Time
}

// SwitchTenantUseCase implementa POST /auth/switch-tenant.
//
// Re-firma el JWT con current_tenant fijado al slug solicitado, siempre
// que la persona tenga membership activa. Es idempotente: llamarlo varias
// veces con el mismo slug devuelve un JWT equivalente (cambia solo iat/exp).
type SwitchTenantUseCase struct {
	deps SwitchTenantDeps
}

// NewSwitchTenantUseCase construye el usecase.
func NewSwitchTenantUseCase(deps SwitchTenantDeps) *SwitchTenantUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &SwitchTenantUseCase{deps: deps}
}

// Execute valida la membership del usuario en el tenant slug y devuelve
// un JWT con current_tenant fijado.
func (uc *SwitchTenantUseCase) Execute(ctx context.Context, subject string, req dto.SwitchTenantRequest) (dto.SwitchTenantResponse, error) {
	slug := strings.TrimSpace(strings.ToLower(req.TenantSlug))
	if slug == "" {
		return dto.SwitchTenantResponse{}, ErrInvalidInput
	}
	id, err := uuid.Parse(subject)
	if err != nil {
		return dto.SwitchTenantResponse{}, ErrInvalidInput
	}

	memberships, err := uc.deps.Users.ListMemberships(ctx, id)
	if err != nil {
		return dto.SwitchTenantResponse{}, fmt.Errorf("%w: list memberships: %w", ErrInternal, err)
	}

	var match *dto.MembershipDTO
	mclaims := make([]jwtsign.MembershipClaim, 0, len(memberships))
	for _, m := range memberships {
		dm := dto.MembershipDTO{
			TenantID:     m.TenantID.String(),
			TenantSlug:   m.TenantSlug,
			TenantName:   m.TenantName,
			LogoURL:      m.LogoURL,
			PrimaryColor: m.PrimaryColor,
			Role:         m.Role,
			Status:       m.Status,
		}
		mclaims = append(mclaims, jwtsign.MembershipClaim{
			TenantID:   dm.TenantID,
			TenantSlug: dm.TenantSlug,
			TenantName: dm.TenantName,
			Role:       dm.Role,
		})
		if m.TenantSlug == slug {
			cp := dm
			match = &cp
		}
	}
	if match == nil {
		return dto.SwitchTenantResponse{}, ErrMembershipMissing
	}

	now := uc.deps.Now()
	access, err := uc.deps.Signer.SignPlatform(
		subject,
		newSessionID(now),
		slug,
		mclaims,
		[]string{"pwd", "switch"},
		accessTokenTTL,
	)
	if err != nil {
		return dto.SwitchTenantResponse{}, fmt.Errorf("%w: sign access: %w", ErrInternal, err)
	}
	return dto.SwitchTenantResponse{
		AccessToken:   access,
		TokenType:     tokenTypeBearer,
		ExpiresIn:     int(accessTokenTTL / time.Second),
		CurrentTenant: *match,
	}, nil
}
