package middleware

import (
	"context"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	dbpkg "github.com/saas-ph/api/internal/platform/db"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// HeaderTenantSlug se mantiene exportado por compatibilidad con clientes
// internos (scripts, tests historicos). En produccion post-Fase 16 ya no
// se consulta — el tenant viene del JWT.
const HeaderTenantSlug = "X-Tenant-Slug"

// tenantLookup es la superficie minima del Registry que requiere el
// middleware. Se define localmente para desacoplar el middleware del
// tipo concreto *db.Registry y poder inyectar fakes en tests sin tocar
// Postgres.
type tenantLookup interface {
	Get(ctx context.Context, slug string) (dbpkg.TenantMetadata, *pgxpool.Pool, error)
}

// TenantResolverConfig agrupa las dependencias del middleware.
//
// Reglas de cableado post-Fase 16 (ADR 0007):
//   - PlatformAuth DEBE correr antes que TenantResolver. Las claims se
//     leen via PlatformAuthFromCtx; si no hay → 412.
//   - Registry es obligatorio. Si es nil el middleware responde 500.
//   - Skip se ejecuta antes de la resolucion. Devolver true significa
//     "este path no tiene tenant" — ej. /health, /superadmin/*, los
//     endpoints centrales /auth/* y /me/*.
//   - Logger es opcional; si es nil se descarta.
type TenantResolverConfig struct {
	Registry tenantLookup
	Skip     func(r *http.Request) bool
	Logger   *slog.Logger
}

// TenantResolver es un middleware chi-compatible que extrae current_tenant
// del JWT validado por PlatformAuth, verifica que el usuario tenga
// membresia activa, lee la metadata del tenant via Registry y la inyecta
// en el contexto.
//
// Errores emitidos (Problem+JSON):
//   - 412 Precondition Failed: claims sin current_tenant (el cliente debe
//     llamar /auth/switch-tenant primero).
//   - 403 Forbidden: el JWT no incluye membresia activa para ese slug,
//     o el tenant esta suspendido.
//   - 404 Not Found: slug bien formado pero desconocido por el Registry.
//   - 500 Internal: Registry/Auth no configurados o lookup fallido.
func TenantResolver(cfg TenantResolverConfig) func(http.Handler) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			if cfg.Registry == nil {
				logger.ErrorContext(r.Context(),
					"tenant_resolver: registry no configurado",
					slog.String("path", r.URL.Path),
				)
				apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
				return
			}

			claims, ok := PlatformAuthFromCtx(r.Context())
			if !ok || claims == nil {
				// 401 antes de 412: si no hubo PlatformAuth no podemos
				// pedir current_tenant — la falla es del cableado del
				// stack, no del cliente.
				apperrors.Write(w, apperrors.Unauthorized("authentication required").WithInstance(r.URL.Path))
				return
			}

			slug := claims.CurrentTenant
			if slug == "" {
				apperrors.Write(w,
					apperrors.New(http.StatusPreconditionFailed, "tenant-not-selected",
						"Tenant Not Selected",
						"current_tenant missing in JWT; call POST /auth/switch-tenant first").
						WithInstance(r.URL.Path))
				return
			}

			// Verificar membresia activa para ese slug en las claims.
			// Defensa contra JWT manipulado (la firma ya se valido pero
			// reforzamos): si el slug no esta en memberships → 403.
			if !hasActiveMembership(claims.Memberships, slug) {
				apperrors.Write(w, apperrors.Forbidden("no active membership in tenant").WithInstance(r.URL.Path))
				return
			}

			meta, pool, err := cfg.Registry.Get(r.Context(), slug)
			if err != nil {
				if stderrors.Is(err, dbpkg.ErrTenantNotFound) {
					apperrors.Write(w, apperrors.NotFound("tenant not found").WithInstance(r.URL.Path))
					return
				}
				logger.ErrorContext(r.Context(),
					"tenant_resolver: lookup fallido",
					slog.String("slug", slug),
					slog.String("path", r.URL.Path),
					slog.String("error", err.Error()),
				)
				apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
				return
			}

			// El Registry solo devuelve tenants activos; pero el campo
			// Status de la metadata no esta expuesto. Si el tenant fue
			// suspendido despues del JWT, el JWT sigue valido hasta exp;
			// el Registry filtra por status='active' en su lookup.
			tenant := &tenantctx.Tenant{
				ID:          meta.ID,
				Slug:        meta.Slug,
				DisplayName: meta.DisplayName,
				Pool:        pool,
			}
			ctx := tenantctx.WithTenant(r.Context(), tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func hasActiveMembership(memberships []jwtsign.MembershipClaim, slug string) bool {
	for _, m := range memberships {
		if m.TenantSlug == slug {
			return true
		}
	}
	return false
}
