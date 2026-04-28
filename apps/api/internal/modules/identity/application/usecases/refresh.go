package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// RefreshDeps agrupa las dependencias del usecase de refresh.
type RefreshDeps struct {
	Users    domain.UserRepository
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// RefreshUseCase implementa POST /auth/refresh.
type RefreshUseCase struct {
	deps RefreshDeps
}

// NewRefreshUseCase construye el usecase. Si Now es nil usa time.Now.
func NewRefreshUseCase(deps RefreshDeps) *RefreshUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &RefreshUseCase{deps: deps}
}

// Execute rota el refresh token: marca el anterior como revocado con
// motivo `rotated`, crea uno nuevo encadenado al previo y devuelve el
// par access+refresh nuevo. Si el refresh recibido ya estaba revocado,
// se considera reuso e invalida toda la cadena (parent + descendientes).
func (uc *RefreshUseCase) Execute(ctx context.Context, req dto.RefreshRequest) (dto.RefreshResponse, error) {
	plain := strings.TrimSpace(req.RefreshToken)
	if plain == "" {
		return dto.RefreshResponse{}, ErrInvalidInput
	}

	hash := HashRefreshToken(plain)
	session, err := uc.deps.Sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return dto.RefreshResponse{}, ErrInvalidRefresh
		}
		return dto.RefreshResponse{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	now := uc.deps.Now()

	if session.IsRevoked() {
		// Si el motivo previo fue "rotated", entonces este token fue
		// rotado legitimamente y ahora alguien intenta reusarlo: es
		// reuso. Revocar toda la cadena.
		if session.RevocationReasonValue() == entities.RevocationReasonRotated {
			_ = uc.deps.Sessions.RevokeChain(ctx, session.ID, entities.RevocationReasonReuseDetected, now)
		}
		return dto.RefreshResponse{}, ErrInvalidRefresh
	}
	if session.IsExpiredAt(now) {
		return dto.RefreshResponse{}, ErrInvalidRefresh
	}

	user, err := uc.deps.Users.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.RefreshResponse{}, ErrInvalidRefresh
		}
		return dto.RefreshResponse{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	if !user.IsActive() {
		return dto.RefreshResponse{}, ErrAccountInactive
	}

	access, refresh, err := IssueSession(ctx, uc.deps.Sessions, uc.deps.Signer, user.ID, session.ID, now)
	if err != nil {
		return dto.RefreshResponse{}, err
	}

	if err := uc.deps.Sessions.Revoke(ctx, session.ID, entities.RevocationReasonRotated, now); err != nil {
		return dto.RefreshResponse{}, fmt.Errorf("%w: revoke previous: %w", ErrInternal, err)
	}

	return dto.RefreshResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(accessTokenTTL / time.Second),
		TokenType:    tokenTypeBearer,
	}, nil
}
