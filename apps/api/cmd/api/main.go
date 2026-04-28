// Command api levanta el servidor HTTP del SaaS Propiedad Horizontal.
//
// Ciclo de vida:
//  1. Carga configuracion desde env.
//  2. Inicializa logger structured (slog).
//  3. Abre el pool del Control Plane (en arranque local sin Postgres,
//     el proceso continua en modo degradado y solo /health responde).
//  4. Construye un Registry de pools por tenant que consulta la tabla
//     `tenants` del Control Plane.
//  5. Cablea middlewares (RequestID, Logging, Recovery, RateLimit,
//     TenantResolver opcional por path).
//  6. Registra rutas: /health, /ready (Control Plane), /tenant/ready
//     (requiere tenant resuelto).
//  7. Arranca el servidor con graceful shutdown ante SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/handlers"
	authzpersistence "github.com/saas-ph/api/internal/modules/authorization/infrastructure/persistence"
	authzhttp "github.com/saas-ph/api/internal/modules/authorization/interfaces/http"
	idpersistence "github.com/saas-ph/api/internal/modules/identity/infrastructure/persistence"
	idhttp "github.com/saas-ph/api/internal/modules/identity/interfaces/http"
	tcpersistence "github.com/saas-ph/api/internal/modules/tenant_config/infrastructure/persistence"
	tchttp "github.com/saas-ph/api/internal/modules/tenant_config/interfaces/http"
	"github.com/saas-ph/api/internal/platform/config"
	"github.com/saas-ph/api/internal/platform/db"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/middleware"
	"github.com/saas-ph/api/internal/version"
)

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	logger := middleware.NewLogger(cfg.Log.Format, cfg.Log.Level)
	logger.Info("api starting",
		slog.String("version", version.Version),
		slog.String("addr", cfg.HTTP.Addr),
	)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	centralPool, err := db.NewPool(rootCtx, db.PoolConfig{
		URL:             cfg.Database.CentralURL,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
	})
	if err != nil {
		logger.Warn("control plane pool unavailable, running degraded",
			slog.String("error", err.Error()))
		centralPool = nil
	} else {
		defer centralPool.Close()
	}

	var registry *db.Registry
	if centralPool != nil {
		registry, err = db.NewRegistry(db.RegistryConfig{
			Lookup: db.LookupFromCentral(centralPool),
			PoolConfig: db.PoolConfig{
				MaxConns:        cfg.Database.MaxConns,
				MinConns:        cfg.Database.MinConns,
				MaxConnLifetime: cfg.Database.MaxConnLifetime,
			},
			CacheTTL:   cfg.Tenant.CacheTTL,
			MaxEntries: cfg.Tenant.CacheSize,
		})
		if err != nil {
			return err
		}
		defer registry.Close()
	}

	signer, err := jwtsign.NewSigner(jwtsign.SignerConfig{
		Issuer:   "ph-saas",
		Audience: "ph-tenant",
	})
	if err != nil {
		return err
	}

	router := buildRouter(logger, cfg, centralPool, registry, signer)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", slog.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown requested", slog.String("signal", sig.String()))
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("api stopped")
	return nil
}

func buildRouter(logger *slog.Logger, cfg config.Config, centralPool *pgxpool.Pool, registry *db.Registry, signer *jwtsign.Signer) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.RateLimit(middleware.RateLimitConfig{
		RequestsPerSecond: 50,
		Burst:             100,
	}))

	// Rutas publicas (sin tenant).
	r.Get("/health", handlers.Health)
	if centralPool != nil {
		r.Get("/ready", handlers.Ready(centralPool))
	}

	// Rutas con tenant resuelto.
	if registry != nil {
		r.Group(func(tr chi.Router) {
			tr.Use(middleware.TenantResolver(middleware.TenantResolverConfig{
				Registry:   registry,
				BaseDomain: cfg.Tenant.BaseDomain,
				Logger:     logger,
				Skip: func(req *http.Request) bool {
					p := req.URL.Path
					return p == "/health" || p == "/ready" || strings.HasPrefix(p, "/superadmin/")
				},
			}))
			tr.Get("/tenant/ready", handlers.TenantReady)

			// Modulo identity.
			idhttp.Mount(tr, idhttp.Dependencies{
				Logger:      logger,
				Signer:      signer,
				UserRepo:    idpersistence.NewUserRepository(),
				SessionRepo: idpersistence.NewSessionRepository(),
				Now:         time.Now,
			})

			// Modulo authorization.
			authzhttp.Mount(tr, authzhttp.Dependencies{
				Logger:         logger,
				RoleRepo:       authzpersistence.NewRoleRepo(),
				PermissionRepo: authzpersistence.NewPermissionRepo(),
				AssignmentRepo: authzpersistence.NewAssignmentRepo(),
				Now:            time.Now,
			})

			// Modulo tenant_config.
			tchttp.Mount(tr, tchttp.Dependencies{
				Logger:       logger,
				SettingsRepo: tcpersistence.NewSettingsRepository(),
				BrandingRepo: tcpersistence.NewBrandingRepository(),
				Now:          time.Now,
			})
		})
	}

	return r
}
