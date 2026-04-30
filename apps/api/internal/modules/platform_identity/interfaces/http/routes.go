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
	Logger      *slog.Logger
	Signer      *jwtsign.Signer
	UserRepo    domain.PlatformUserRepository
	SessionRepo domain.SessionRepository
	DeviceRepo  domain.PushDeviceRepository
	Now         func() time.Time
}

// Mount registra los endpoints de identidad de plataforma en el router
// chi recibido. Estos endpoints NO viven detras del tenant_resolver
// porque la identidad es global (DB central).
//
// Endpoints en esta version:
//   - POST /auth/login
//   - POST /auth/switch-tenant   (auth)
//   - GET  /me                   (auth)
//   - GET  /me/memberships       (auth)
//
// Pendientes de implementacion en proximas iteraciones:
//   - POST /auth/mfa/verify
//   - POST /auth/refresh
//   - POST /auth/logout
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
		signer: deps.Signer,
		loginUC: usecases.NewLoginUseCase(usecases.LoginDeps{
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
		}),
		mfaVerifyUC: usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
			Users:    deps.UserRepo,
			Sessions: deps.SessionRepo,
			Signer:   deps.Signer,
			Now:      now,
		}),
		meUC: usecases.NewMeUseCase(usecases.MeDeps{Users: deps.UserRepo}),
		listMembershipsUC: usecases.NewListMembershipsUseCase(usecases.ListMembershipsDeps{
			Users: deps.UserRepo,
		}),
		switchTenantUC: usecases.NewSwitchTenantUseCase(usecases.SwitchTenantDeps{
			Users:  deps.UserRepo,
			Signer: deps.Signer,
			Now:    now,
		}),
		registerDeviceUC: usecases.NewRegisterPushDeviceUseCase(usecases.RegisterPushDeviceDeps{
			Devices: deps.DeviceRepo,
		}),
		removeDeviceUC: usecases.NewRemovePushDeviceUseCase(usecases.RemovePushDeviceDeps{
			Devices: deps.DeviceRepo,
		}),
	}

	r.Route("/auth", func(ar chi.Router) {
		ar.Post("/login", h.login)
		ar.Post("/mfa/verify", h.mfaVerify)
		ar.Post("/refresh", h.refresh)
		ar.Post("/logout", h.logout)
		ar.Group(func(pr chi.Router) {
			pr.Use(h.authMiddleware)
			pr.Post("/switch-tenant", h.switchTenant)
		})
	})

	r.Group(func(pr chi.Router) {
		pr.Use(h.authMiddleware)
		pr.Get("/me", h.me)
		pr.Get("/me/memberships", h.memberships)
		pr.Post("/me/push-devices", h.registerPushDevice)
		pr.Delete("/me/push-devices/{deviceID}", h.removePushDevice)
	})
}
