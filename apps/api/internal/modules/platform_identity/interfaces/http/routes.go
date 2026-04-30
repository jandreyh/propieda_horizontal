package http

import (
	"io"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/usecases"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// Dependencies agrupa las dependencias que el orquestador (cmd/api)
// inyecta al modulo platform_identity al montar sus rutas.
//
// Logger es opcional (si es nil se descarta).
// Signer es obligatorio (firma JWT post-Fase 16 con SignPlatform).
// UserRepo es obligatorio.
// Now es opcional; default time.Now.
type Dependencies struct {
	Logger   *slog.Logger
	Signer   *jwtsign.Signer
	UserRepo domain.PlatformUserRepository
	Now      func() time.Time
}

// Mount registra los endpoints publicos de identidad de plataforma en el
// router chi recibido. Estos endpoints NO viven detras del tenant_resolver
// porque la identidad es global (DB central).
//
// Endpoints en esta version:
//   - POST /auth/login
//
// Pendientes de implementacion en proximas iteraciones:
//   - POST /auth/mfa/verify
//   - POST /auth/refresh
//   - POST /auth/logout
//   - POST /auth/switch-tenant
//   - GET  /me
//   - GET  /me/memberships
//   - POST /me/push-devices
//   - DELETE /me/push-devices/{id}
func Mount(r chi.Router, deps Dependencies) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}

	h := &handlers{
		logger: logger,
		loginUC: usecases.NewLoginUseCase(usecases.LoginDeps{
			Users:  deps.UserRepo,
			Signer: deps.Signer,
			Now:    now,
		}),
	}

	r.Route("/auth", func(ar chi.Router) {
		ar.Post("/login", h.login)
	})
}
