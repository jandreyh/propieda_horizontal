package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// LogoutDeps agrupa las dependencias del usecase de logout.
type LogoutDeps struct {
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// LogoutUseCase implementa POST /auth/logout.
type LogoutUseCase struct {
	deps LogoutDeps
}

// NewLogoutUseCase construye el usecase. Si Now es nil usa time.Now.
func NewLogoutUseCase(deps LogoutDeps) *LogoutUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &LogoutUseCase{deps: deps}
}

// Execute revoca la sesion identificada por el access token recibido.
// El access token JWT contiene la claim `sid` con el id de la sesion;
// no necesitamos el refresh para revocar.
func (uc *LogoutUseCase) Execute(ctx context.Context, accessToken string) error {
	tok := strings.TrimSpace(accessToken)
	if tok == "" {
		return ErrInvalidInput
	}
	claims, err := uc.deps.Signer.Verify(tok)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRefresh, err)
	}
	if claims.SessionID == "" || claims.SessionID == PreAuthSessionMarker {
		return ErrInvalidInput
	}
	now := uc.deps.Now()
	if err := uc.deps.Sessions.Revoke(ctx, claims.SessionID, entities.RevocationReasonLogout, now); err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			// idempotente: si ya no existe, consideramos logout OK.
			return nil
		}
		return fmt.Errorf("%w: %w", ErrInternal, err)
	}
	return nil
}
