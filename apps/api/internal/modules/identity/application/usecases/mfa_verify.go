package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// MFAVerifyDeps agrupa las dependencias del usecase de MFA verify.
type MFAVerifyDeps struct {
	Users    domain.UserRepository
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// MFAVerifyUseCase implementa POST /auth/mfa/verify.
type MFAVerifyUseCase struct {
	deps MFAVerifyDeps
}

// NewMFAVerifyUseCase construye el usecase. Si Now es nil usa time.Now.
func NewMFAVerifyUseCase(deps MFAVerifyDeps) *MFAVerifyUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &MFAVerifyUseCase{deps: deps}
}

// Execute valida el pre_auth_token y el codigo TOTP. Si ambos son
// correctos, emite una sesion completa (access + refresh).
func (uc *MFAVerifyUseCase) Execute(ctx context.Context, req dto.MFAVerifyRequest) (dto.MFAVerifyResponse, error) {
	if strings.TrimSpace(req.PreAuthToken) == "" || strings.TrimSpace(req.Code) == "" {
		return dto.MFAVerifyResponse{}, ErrInvalidInput
	}
	claims, err := uc.deps.Signer.Verify(req.PreAuthToken)
	if err != nil {
		return dto.MFAVerifyResponse{}, fmt.Errorf("%w: %w", ErrInvalidPreAuth, err)
	}
	if claims.SessionID != PreAuthSessionMarker || !hasRole(claims.Roles, PreAuthRole) {
		return dto.MFAVerifyResponse{}, ErrInvalidPreAuth
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return dto.MFAVerifyResponse{}, ErrInvalidPreAuth
	}

	user, err := uc.deps.Users.GetByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.MFAVerifyResponse{}, ErrInvalidPreAuth
		}
		return dto.MFAVerifyResponse{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	if !user.IsActive() {
		return dto.MFAVerifyResponse{}, ErrAccountInactive
	}
	if !user.HasMFA() {
		return dto.MFAVerifyResponse{}, ErrInvalidPreAuth
	}

	now := uc.deps.Now()
	if !totp.Validate(req.Code, *user.MFASecret) {
		return dto.MFAVerifyResponse{}, ErrInvalidMFACode
	}

	access, refresh, err := IssueSession(ctx, uc.deps.Sessions, uc.deps.Signer, user.ID, "", now)
	if err != nil {
		return dto.MFAVerifyResponse{}, err
	}

	if err := uc.deps.Users.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
		return dto.MFAVerifyResponse{}, fmt.Errorf("%w: update last_login: %w", ErrInternal, err)
	}

	return dto.MFAVerifyResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(accessTokenTTL / time.Second),
		TokenType:    tokenTypeBearer,
	}, nil
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}
