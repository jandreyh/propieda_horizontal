package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// ErrInvalidRefresh se emite cuando el refresh token no resuelve a una
// sesion activa.
var ErrInvalidRefresh = errors.New("platform_identity: invalid refresh token")

// RefreshDeps son las dependencias del usecase de refresh.
type RefreshDeps struct {
	Users    domain.PlatformUserRepository
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// RefreshUseCase implementa POST /auth/refresh con rotacion del refresh
// token. Al refrescar:
//  1. Hashea el refresh recibido y busca la sesion activa.
//  2. Revoca la sesion actual (refresh token rotation).
//  3. Crea una sesion nueva con un refresh token nuevo.
//  4. Firma un access token nuevo (con memberships actualizados).
type RefreshUseCase struct {
	deps RefreshDeps
}

// NewRefreshUseCase construye el usecase.
func NewRefreshUseCase(deps RefreshDeps) *RefreshUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &RefreshUseCase{deps: deps}
}

// Execute valida y rota el refresh token.
func (uc *RefreshUseCase) Execute(ctx context.Context, req dto.RefreshRequest) (dto.RefreshResponse, error) {
	plain := strings.TrimSpace(req.RefreshToken)
	if plain == "" {
		return dto.RefreshResponse{}, ErrInvalidInput
	}
	hash := HashRefreshToken(plain)
	session, err := uc.deps.Sessions.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return dto.RefreshResponse{}, ErrInvalidRefresh
		}
		return dto.RefreshResponse{}, fmt.Errorf("%w: find session: %w", ErrInternal, err)
	}
	now := uc.deps.Now()
	if !session.IsActive(now) {
		return dto.RefreshResponse{}, ErrInvalidRefresh
	}

	// Revoca la sesion actual antes de emitir la nueva.
	if err := uc.deps.Sessions.Revoke(ctx, session.ID, "rotated"); err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("%w: revoke previous: %w", ErrInternal, err)
	}

	user, err := uc.deps.Users.FindByID(ctx, session.PlatformUserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.RefreshResponse{}, ErrInvalidRefresh
		}
		return dto.RefreshResponse{}, fmt.Errorf("%w: find user: %w", ErrInternal, err)
	}
	if !user.IsActive() {
		return dto.RefreshResponse{}, ErrAccountInactive
	}

	memberships, err := uc.deps.Users.ListMemberships(ctx, user.ID)
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("%w: list memberships: %w", ErrInternal, err)
	}
	mclaims := make([]jwtsign.MembershipClaim, 0, len(memberships))
	for _, m := range memberships {
		mclaims = append(mclaims, jwtsign.MembershipClaim{
			TenantID:   m.TenantID.String(),
			TenantSlug: m.TenantSlug,
			TenantName: m.TenantName,
			Role:       m.Role,
		})
	}

	plainNew, hashNew, err := generateRefreshToken()
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("%w: gen refresh: %w", ErrInternal, err)
	}
	newSession, err := uc.deps.Sessions.Create(ctx, user.ID, hashNew, session.UserAgent, now.Add(refreshTokenTTL))
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("%w: create session: %w", ErrInternal, err)
	}
	access, err := uc.deps.Signer.SignPlatform(
		user.ID.String(),
		newSession.ID.String(),
		"",
		mclaims,
		[]string{"pwd", "refresh"},
		accessTokenTTL,
	)
	if err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("%w: sign: %w", ErrInternal, err)
	}

	return dto.RefreshResponse{
		AccessToken:  access,
		RefreshToken: plainNew,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTokenTTL / time.Second),
	}, nil
}

// LogoutDeps son las dependencias del usecase de logout.
type LogoutDeps struct {
	Sessions domain.SessionRepository
}

// LogoutUseCase implementa POST /auth/logout. Recibe el refresh token
// para revocar exactamente esa sesion.
type LogoutUseCase struct {
	deps LogoutDeps
}

// NewLogoutUseCase construye el usecase.
func NewLogoutUseCase(deps LogoutDeps) *LogoutUseCase {
	return &LogoutUseCase{deps: deps}
}

// Execute revoca la sesion identificada por el refresh token. Si el
// token no existe o ya estaba revocado, devuelve nil silenciosamente
// (logout es idempotente — no fugamos info al cliente).
func (uc *LogoutUseCase) Execute(ctx context.Context, refresh string) error {
	plain := strings.TrimSpace(refresh)
	if plain == "" {
		return ErrInvalidInput
	}
	hash := HashRefreshToken(plain)
	session, err := uc.deps.Sessions.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil // idempotente
		}
		return fmt.Errorf("%w: find session: %w", ErrInternal, err)
	}
	if err := uc.deps.Sessions.Revoke(ctx, session.ID, "logout"); err != nil {
		return fmt.Errorf("%w: revoke: %w", ErrInternal, err)
	}
	return nil
}
