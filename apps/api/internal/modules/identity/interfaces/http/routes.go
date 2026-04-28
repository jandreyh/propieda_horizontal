package http

import (
	"io"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/identity/application/usecases"
	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// Dependencies agrupa las dependencias que el orquestador (cmd/api)
// inyecta al modulo identity al montar sus rutas.
//
// Logger es opcional (si es nil se descarta).
// Signer es obligatorio.
// UserRepo / SessionRepo son obligatorios.
// Now es opcional; default time.Now.
type Dependencies struct {
	Logger      *slog.Logger
	Signer      *jwtsign.Signer
	UserRepo    domain.UserRepository
	SessionRepo domain.SessionRepository
	Now         func() time.Time
}

// Mount registra los endpoints del modulo en el router chi recibido.
//
// Endpoints publicos:
//   - POST /auth/login
//   - POST /auth/mfa/verify
//   - POST /auth/refresh
//
// Endpoints autenticados (Bearer JWT firmado por Signer):
//   - POST /auth/logout
//   - GET  /me
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
		signer: deps.Signer,
		loginUC: usecases.NewLoginUseCase(usecases.LoginDeps{
			Users:    deps.UserRepo,
			Sessions: deps.SessionRepo,
			Signer:   deps.Signer,
			Now:      now,
		}),
		mfaVerifyUC: usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
			Users:    deps.UserRepo,
			Sessions: deps.SessionRepo,
			Signer:   deps.Signer,
			Now:      now,
		}),
		refreshUC: usecases.NewRefreshUseCase(usecases.RefreshDeps{
			Users:    deps.UserRepo,
			Sessions: deps.SessionRepo,
			Signer:   deps.Signer,
			Now:      now,
		}),
		logoutUC: usecases.NewLogoutUseCase(usecases.LogoutDeps{
			Sessions: deps.SessionRepo,
			Signer:   deps.Signer,
			Now:      now,
		}),
		meUC: usecases.NewMeUseCase(usecases.MeDeps{Users: deps.UserRepo}),
	}

	r.Route("/auth", func(ar chi.Router) {
		ar.Post("/login", h.login)
		ar.Post("/mfa/verify", h.mfaVerify)
		ar.Post("/refresh", h.refresh)
		ar.Group(func(pr chi.Router) {
			pr.Use(h.authMiddleware)
			pr.Post("/logout", h.logout)
		})
	})
	r.Group(func(pr chi.Router) {
		pr.Use(h.authMiddleware)
		pr.Get("/me", h.me)
	})
}
