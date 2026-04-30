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
	accesspersistence "github.com/saas-ph/api/internal/modules/access_control/infrastructure/persistence"
	accesshttp "github.com/saas-ph/api/internal/modules/access_control/interfaces/http"
	annpersistence "github.com/saas-ph/api/internal/modules/announcements/infrastructure/persistence"
	annhttp "github.com/saas-ph/api/internal/modules/announcements/interfaces/http"
	asmpersistence "github.com/saas-ph/api/internal/modules/assemblies/infrastructure/persistence"
	asmhttp "github.com/saas-ph/api/internal/modules/assemblies/interfaces/http"
	authzpersistence "github.com/saas-ph/api/internal/modules/authorization/infrastructure/persistence"
	authzhttp "github.com/saas-ph/api/internal/modules/authorization/interfaces/http"
	finpersistence "github.com/saas-ph/api/internal/modules/finance/infrastructure/persistence"
	finhttp "github.com/saas-ph/api/internal/modules/finance/interfaces/http"
	incpersistence "github.com/saas-ph/api/internal/modules/incidents/infrastructure/persistence"
	inchttp "github.com/saas-ph/api/internal/modules/incidents/interfaces/http"
	notifpersistence "github.com/saas-ph/api/internal/modules/notifications/infrastructure/persistence"
	notifhttp "github.com/saas-ph/api/internal/modules/notifications/interfaces/http"
	pkgusecases "github.com/saas-ph/api/internal/modules/packages/application/usecases"
	pkgpersistence "github.com/saas-ph/api/internal/modules/packages/infrastructure/persistence"
	pkghttp "github.com/saas-ph/api/internal/modules/packages/interfaces/http"
	parkingpersistence "github.com/saas-ph/api/internal/modules/parking/infrastructure/persistence"
	parkinghttp "github.com/saas-ph/api/internal/modules/parking/interfaces/http"
	penpersistence "github.com/saas-ph/api/internal/modules/penalties/infrastructure/persistence"
	penhttp "github.com/saas-ph/api/internal/modules/penalties/interfaces/http"
	peoplepersistence "github.com/saas-ph/api/internal/modules/people/infrastructure/persistence"
	peoplehttp "github.com/saas-ph/api/internal/modules/people/interfaces/http"
	platformidpersistence "github.com/saas-ph/api/internal/modules/platform_identity/infrastructure/persistence"
	platformidhttp "github.com/saas-ph/api/internal/modules/platform_identity/interfaces/http"
	pqrspersistence "github.com/saas-ph/api/internal/modules/pqrs/infrastructure/persistence"
	pqrshttp "github.com/saas-ph/api/internal/modules/pqrs/interfaces/http"
	"github.com/saas-ph/api/internal/modules/provisioning"
	respersistence "github.com/saas-ph/api/internal/modules/reservations/infrastructure/persistence"
	reshttp "github.com/saas-ph/api/internal/modules/reservations/interfaces/http"
	rspersistence "github.com/saas-ph/api/internal/modules/residential_structure/infrastructure/persistence"
	rshttp "github.com/saas-ph/api/internal/modules/residential_structure/interfaces/http"
	superadminhttp "github.com/saas-ph/api/internal/modules/superadmin/interfaces/http"
	tcpersistence "github.com/saas-ph/api/internal/modules/tenant_config/infrastructure/persistence"
	tchttp "github.com/saas-ph/api/internal/modules/tenant_config/interfaces/http"
	tmpersistence "github.com/saas-ph/api/internal/modules/tenant_members/infrastructure/persistence"
	tmhttp "github.com/saas-ph/api/internal/modules/tenant_members/interfaces/http"
	unitspersistence "github.com/saas-ph/api/internal/modules/units/infrastructure/persistence"
	unitshttp "github.com/saas-ph/api/internal/modules/units/interfaces/http"
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

	// Modulo platform_identity (post-Fase 16 / ADR 0007). Vive en la DB
	// central y NO se monta detras de tenant_resolver: la identidad es
	// global y el current_tenant se selecciona via /auth/switch-tenant.
	if centralPool != nil {
		platformidhttp.Mount(r, platformidhttp.Dependencies{
			Logger:      logger,
			Signer:      signer,
			UserRepo:    platformidpersistence.NewPlatformUserRepository(centralPool),
			SessionRepo: platformidpersistence.NewSessionRepository(centralPool),
			DeviceRepo:  platformidpersistence.NewPushDeviceRepository(centralPool),
			Now:         time.Now,
		})
	}

	// Modulo superadmin + provisioning. Tambien fuera del tenant_resolver:
	// el superadmin opera contra la DB central. La autorizacion se hace
	// inline en el modulo (rol platform_superadmin en el JWT).
	//
	// PROVISIONING_MAINTENANCE_URL y PROVISIONING_TENANT_URL_TEMPLATE
	// son los unicos parametros adicionales requeridos. Si falta alguno
	// se omite el wiring (los endpoints no estaran disponibles).
	if centralPool != nil {
		maintURL := os.Getenv("PROVISIONING_MAINTENANCE_URL")
		urlTpl := os.Getenv("PROVISIONING_TENANT_URL_TEMPLATE")
		migPath := os.Getenv("PROVISIONING_TENANT_MIGRATIONS_PATH")
		if maintURL != "" && urlTpl != "" && migPath != "" {
			prov := provisioning.New(provisioning.Config{
				CentralPool:        centralPool,
				MaintenanceURL:     maintURL,
				AdminURLTemplate:   urlTpl,
				MigrationsPathFile: migPath,
			})
			superadminhttp.Mount(r, superadminhttp.Dependencies{
				Logger:      logger,
				CentralPool: centralPool,
				Provisioner: prov,
			})
		}
	}

	// Rutas con tenant resuelto.
	if registry != nil {
		r.Group(func(tr chi.Router) {
			tr.Use(middleware.PlatformAuth(middleware.PlatformAuthConfig{
				Signer: signer,
				Skip: func(req *http.Request) bool {
					p := req.URL.Path
					return p == "/health" || p == "/ready" || strings.HasPrefix(p, "/superadmin/")
				},
			}))
			tr.Use(middleware.TenantResolver(middleware.TenantResolverConfig{
				Registry: registry,
				Logger:   logger,
				Skip: func(req *http.Request) bool {
					p := req.URL.Path
					return p == "/health" || p == "/ready" || strings.HasPrefix(p, "/superadmin/")
				},
			}))
			tr.Get("/tenant/ready", handlers.TenantReady)

			// Modulo tenant_members (Fase 16): vinculacion de personas
			// al conjunto via public_code. Vive bajo tenant_resolver
			// porque opera contra tenant_user_links del tenant.
			tmhttp.Mount(tr, tmhttp.Dependencies{
				Logger:   logger,
				Links:    tmpersistence.NewLinkRepository(),
				Enricher: tmpersistence.NewEnricherRepository(centralPool),
			})

			// Modulo identity LEGACY — superseded por platform_identity
			// (ADR 0007). Sus endpoints (/auth/*) los provee ahora
			// platform_identity desde el router raiz, fuera del
			// tenant_resolver. Las queries de identity legacy hacian
			// JOIN con la tabla `users` que desaparece post-Fase 16.
			//
			// idhttp.Mount(tr, idhttp.Dependencies{...})  // disabled

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

			// Modulo residential_structure.
			rshttp.Mount(tr, rshttp.Dependencies{
				Logger:        logger,
				StructureRepo: rspersistence.NewStructureRepository(),
				Now:           time.Now,
			})

			// Modulo units.
			unitshttp.Mount(tr, unitshttp.Dependencies{
				Logger:        logger,
				UnitRepo:      unitspersistence.NewUnitRepo(),
				OwnerRepo:     unitspersistence.NewOwnerRepo(),
				OccupancyRepo: unitspersistence.NewOccupancyRepo(),
				PeopleRepo:    unitspersistence.NewPeopleByUnitRepo(),
				Now:           time.Now,
			})

			// Modulo people (vehiculos).
			peoplehttp.Mount(tr, peoplehttp.Dependencies{
				Logger:         logger,
				VehicleRepo:    peoplepersistence.NewVehicleRepository(),
				AssignmentRepo: peoplepersistence.NewAssignmentRepository(),
				Now:            time.Now,
			})

			// Modulo access_control (porteria/visitas).
			accesshttp.Mount(tr, accesshttp.Dependencies{
				Logger:        logger,
				BlacklistRepo: accesspersistence.NewBlacklistRepository(),
				PreRegRepo:    accesspersistence.NewPreRegistrationRepository(),
				EntryRepo:     accesspersistence.NewVisitorEntryRepository(),
				Now:           time.Now,
			})

			// Modulo packages (correspondencia/paqueteria).
			pkghttp.Mount(tr, pkghttp.Dependencies{
				Logger:      logger,
				Packages:    pkgpersistence.NewPackageRepository(),
				Categories:  pkgpersistence.NewCategoryRepository(),
				Deliveries:  pkgpersistence.NewDeliveryRepository(),
				Outbox:      pkgpersistence.NewOutboxRepository(),
				TxRunner:    pkgpersistence.NewTenantTxRunner(),
				Idempotency: pkgusecases.NewIdempotencyCache(24*time.Hour, time.Now),
				Now:         time.Now,
			})

			// Modulo announcements (tablero).
			annhttp.Mount(tr, annhttp.Dependencies{
				Logger:            logger,
				AnnouncementsRepo: annpersistence.NewAnnouncementRepository(),
				AudiencesRepo:     annpersistence.NewAudienceRepository(),
				AcksRepo:          annpersistence.NewAckRepository(),
				TxRunner:          annpersistence.NewTenantTxRunner(),
				Now:               time.Now,
			})

			// Modulo parking (parqueaderos).
			parkinghttp.Mount(tr, parkinghttp.Dependencies{
				Logger:       logger,
				Spaces:       parkingpersistence.NewSpaceRepository(),
				Assignments:  parkingpersistence.NewAssignmentRepository(),
				History:      parkingpersistence.NewAssignmentHistoryRepository(),
				Reservations: parkingpersistence.NewVisitorReservationRepository(),
				Lotteries:    parkingpersistence.NewLotteryRunRepository(),
				Results:      parkingpersistence.NewLotteryResultRepository(),
				Outbox:       parkingpersistence.NewOutboxRepository(),
				TxRunner:     parkingpersistence.NewTenantTxRunner(),
				Now:          time.Now,
			})

			// Modulo finance (financiero).
			finhttp.Mount(tr, finhttp.Dependencies{
				Logger:          logger,
				Accounts:        finpersistence.NewChartOfAccountsRepository(),
				CostCenters:     finpersistence.NewCostCenterRepository(),
				BillingAccounts: finpersistence.NewBillingAccountRepository(),
				Charges:         finpersistence.NewChargeRepository(),
				Payments:        finpersistence.NewPaymentRepository(),
				Allocations:     finpersistence.NewPaymentAllocationRepository(),
				Reversals:       finpersistence.NewPaymentReversalRepository(),
				Closures:        finpersistence.NewPeriodClosureRepository(),
				Webhooks:        finpersistence.NewWebhookIdempotencyRepository(),
				Outbox:          finpersistence.NewOutboxRepository(),
				TxRunner:        finpersistence.NewTenantTxRunner(),
				Now:             time.Now,
			})

			// Modulo reservations (zonas comunes).
			reshttp.Mount(tr, reshttp.Dependencies{
				Logger:       logger,
				CommonAreas:  respersistence.NewCommonAreaRepository(),
				Blackouts:    respersistence.NewBlackoutRepository(),
				Reservations: respersistence.NewReservationRepository(),
				History:      respersistence.NewStatusHistoryRepository(),
				Outbox:       respersistence.NewOutboxRepository(),
				TxRunner:     respersistence.NewTenantTxRunner(),
				Now:          time.Now,
			})

			// Modulo assemblies (asambleas).
			asmhttp.Mount(tr, asmhttp.Dependencies{
				Logger:      logger,
				Assemblies:  asmpersistence.NewAssemblyRepository(),
				Calls:       asmpersistence.NewCallRepository(),
				Attendances: asmpersistence.NewAttendanceRepository(),
				Proxies:     asmpersistence.NewProxyRepository(),
				Motions:     asmpersistence.NewMotionRepository(),
				Votes:       asmpersistence.NewVoteRepository(),
				Evidence:    asmpersistence.NewVoteEvidenceRepository(),
				Acts:        asmpersistence.NewActRepository(),
				Signatures:  asmpersistence.NewActSignatureRepository(),
				Outbox:      asmpersistence.NewOutboxRepository(),
				TxRunner:    asmpersistence.NewTenantTxRunner(),
				Now:         time.Now,
				MaxProxies:  1,
			})

			// Modulo incidents (incidentes).
			inchttp.Mount(tr, inchttp.Dependencies{
				Logger:      logger,
				Incidents:   incpersistence.NewIncidentRepository(),
				Attachments: incpersistence.NewAttachmentRepository(),
				History:     incpersistence.NewStatusHistoryRepository(),
				Assignments: incpersistence.NewIncidentAssignmentRepository(),
				Outbox:      incpersistence.NewOutboxRepository(),
				TxRunner:    incpersistence.NewTenantTxRunner(),
				Now:         time.Now,
			})

			// Modulo penalties (multas/sanciones).
			penhttp.Mount(tr, penhttp.Dependencies{
				Logger:    logger,
				Catalog:   penpersistence.NewCatalogRepository(),
				Penalties: penpersistence.NewPenaltyRepository(),
				Appeals:   penpersistence.NewAppealRepository(),
				History:   penpersistence.NewStatusHistoryRepository(),
				Outbox:    penpersistence.NewOutboxRepository(),
				TxRunner:  penpersistence.NewTenantTxRunner(),
				Now:       time.Now,
			})

			// Modulo pqrs (peticiones/quejas/reclamos).
			pqrshttp.Mount(tr, pqrshttp.Dependencies{
				Logger:     logger,
				Categories: pqrspersistence.NewCategoryRepository(),
				Tickets:    pqrspersistence.NewTicketRepository(),
				Responses:  pqrspersistence.NewResponseRepository(),
				History:    pqrspersistence.NewStatusHistoryRepository(),
				Outbox:     pqrspersistence.NewOutboxRepository(),
				TxRunner:   pqrspersistence.NewTenantTxRunner(),
				Now:        time.Now,
			})

			// Modulo notifications (multicanal).
			notifhttp.Mount(tr, notifhttp.Dependencies{
				Logger:          logger,
				Templates:       notifpersistence.NewTemplateRepository(),
				Preferences:     notifpersistence.NewPreferenceRepository(),
				Consents:        notifpersistence.NewConsentRepository(),
				PushTokens:      notifpersistence.NewPushTokenRepository(),
				ProviderConfigs: notifpersistence.NewProviderConfigRepository(),
				Outbox:          notifpersistence.NewOutboxRepository(),
				Deliveries:      notifpersistence.NewDeliveryRepository(),
				TxRunner:        notifpersistence.NewTenantTxRunner(),
				Now:             time.Now,
			})
		})
	}

	return r
}
