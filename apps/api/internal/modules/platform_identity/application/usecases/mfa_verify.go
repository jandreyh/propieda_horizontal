package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// ErrInvalidPreAuth se emite cuando el pre_auth_token es invalido,
// expirado o no fue producido por la primera fase del login.
var ErrInvalidPreAuth = errors.New("platform_identity: invalid pre-auth token")

// ErrInvalidMFACode se emite cuando el codigo TOTP no valida.
var ErrInvalidMFACode = errors.New("platform_identity: invalid mfa code")

// MFAVerifyDeps son las dependencias del usecase.
type MFAVerifyDeps struct {
	Users    domain.PlatformUserRepository
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// MFAVerifyUseCase implementa POST /auth/mfa/verify: recibe pre_auth_token
// + codigo TOTP, y emite access + refresh token reales si ambos son
// validos.
type MFAVerifyUseCase struct {
	deps MFAVerifyDeps
}

// NewMFAVerifyUseCase construye el usecase.
func NewMFAVerifyUseCase(deps MFAVerifyDeps) *MFAVerifyUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &MFAVerifyUseCase{deps: deps}
}

// Execute valida pre_auth + TOTP y emite tokens reales.
func (uc *MFAVerifyUseCase) Execute(ctx context.Context, req dto.MFAVerifyRequest) (dto.LoginResponse, error) {
	preAuth := strings.TrimSpace(req.PreAuthToken)
	code := strings.TrimSpace(req.Code)
	if preAuth == "" || code == "" {
		return dto.LoginResponse{}, ErrInvalidInput
	}

	claims, err := uc.deps.Signer.Verify(preAuth)
	if err != nil {
		return dto.LoginResponse{}, ErrInvalidPreAuth
	}
	if claims.SessionID != PreAuthSessionMarker {
		return dto.LoginResponse{}, ErrInvalidPreAuth
	}
	if !slices.Contains(claims.Roles, PreAuthRole) {
		return dto.LoginResponse{}, ErrInvalidPreAuth
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return dto.LoginResponse{}, ErrInvalidPreAuth
	}
	user, err := uc.deps.Users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.LoginResponse{}, ErrInvalidPreAuth
		}
		return dto.LoginResponse{}, fmt.Errorf("%w: find user: %w", ErrInternal, err)
	}
	if !user.IsActive() {
		return dto.LoginResponse{}, ErrAccountInactive
	}
	if user.MFASecret == nil || *user.MFASecret == "" {
		// El JWT decia MFA pero el usuario no tiene secret — estado
		// invalido; tratar como pre-auth invalido.
		return dto.LoginResponse{}, ErrInvalidPreAuth
	}
	if !totp.Validate(code, *user.MFASecret) {
		return dto.LoginResponse{}, ErrInvalidMFACode
	}

	now := uc.deps.Now()
	if err := uc.deps.Users.MarkLoginSuccess(ctx, user.ID, now); err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: mark login: %w", ErrInternal, err)
	}

	memberships, err := uc.deps.Users.ListMemberships(ctx, user.ID)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: list memberships: %w", ErrInternal, err)
	}
	mclaims := make([]jwtsign.MembershipClaim, 0, len(memberships))
	mdtos := make([]dto.MembershipDTO, 0, len(memberships))
	for _, m := range memberships {
		mclaims = append(mclaims, jwtsign.MembershipClaim{
			TenantID:   m.TenantID.String(),
			TenantSlug: m.TenantSlug,
			TenantName: m.TenantName,
			Role:       m.Role,
		})
		mdtos = append(mdtos, dto.MembershipDTO{
			TenantID:     m.TenantID.String(),
			TenantSlug:   m.TenantSlug,
			TenantName:   m.TenantName,
			LogoURL:      m.LogoURL,
			PrimaryColor: m.PrimaryColor,
			Role:         m.Role,
			Status:       m.Status,
		})
	}

	var sessionID, refreshPlain string
	if uc.deps.Sessions != nil {
		plain, hash, err := generateRefreshToken()
		if err != nil {
			return dto.LoginResponse{}, fmt.Errorf("%w: gen refresh: %w", ErrInternal, err)
		}
		s, err := uc.deps.Sessions.Create(ctx, user.ID, hash, nil, now.Add(refreshTokenTTL))
		if err != nil {
			return dto.LoginResponse{}, fmt.Errorf("%w: create session: %w", ErrInternal, err)
		}
		sessionID = s.ID.String()
		refreshPlain = plain
	} else {
		sessionID = fmt.Sprintf("plat-%d", now.UnixNano())
	}

	access, err := uc.deps.Signer.SignPlatform(
		user.ID.String(),
		sessionID,
		"",
		mclaims,
		[]string{"pwd", "mfa"},
		accessTokenTTL,
	)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: sign access: %w", ErrInternal, err)
	}
	return dto.LoginResponse{
		AccessToken:  access,
		RefreshToken: refreshPlain,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTokenTTL / time.Second),
		Memberships:  mdtos,
		NeedsTenant:  len(mdtos) != 1,
	}, nil
}
