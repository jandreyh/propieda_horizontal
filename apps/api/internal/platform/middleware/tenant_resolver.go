package middleware

import (
	"context"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	dbpkg "github.com/saas-ph/api/internal/platform/db"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// HeaderTenantSlug es el nombre canonico del header HTTP usado por
// clientes (movil, scripts internos) para indicar explicitamente el
// tenant cuando no es viable inferirlo del subdominio.
const HeaderTenantSlug = "X-Tenant-Slug"

// tenantSlugMaxLen es el limite superior del slug de tenant. Esta
// alineado con el limite practico de un label DNS (RFC 1035 sec. 2.3.4).
const tenantSlugMaxLen = 63

// tenantSlugPattern restringe el slug a labels DNS-friendly minusculas:
// debe empezar con [a-z0-9] y aceptar guiones internos no consecutivos
// segun el regex `^[a-z0-9](-?[a-z0-9])*$`.
var tenantSlugPattern = regexp.MustCompile(`^[a-z0-9](-?[a-z0-9])*$`)

// tenantLookup es la superficie minima del Registry que requiere el
// middleware. Se define localmente para desacoplar el middleware del
// tipo concreto *db.Registry y poder inyectar fakes en tests sin tocar
// Postgres.
type tenantLookup interface {
	Get(ctx context.Context, slug string) (dbpkg.TenantMetadata, *pgxpool.Pool, error)
}

// TenantResolverConfig agrupa las dependencias del middleware
// TenantResolver.
//
// Reglas de cableado:
//   - Registry es obligatorio. Si es nil el middleware responde 500 a
//     cada request operativo (no se panickea para no matar el proceso).
//   - BaseDomain es obligatorio cuando se quiere resolver por subdominio.
//     Cualquier cliente que entre por header puede operar aunque
//     BaseDomain este vacio.
//   - HeaderName por defecto es "X-Tenant-Slug" (HeaderTenantSlug).
//   - Skip se ejecuta antes de la resolucion. Devolver true significa
//     "este path no tiene tenant" — ej. /health, /superadmin/*.
//   - Logger es opcional; si es nil se descarta.
type TenantResolverConfig struct {
	// Registry resuelve metadata + pool del tenant a partir de su slug.
	Registry tenantLookup
	// BaseDomain es el dominio base donde los subdominios identifican al
	// tenant (ej. "ph.localhost"; entonces "acacias.ph.localhost" -> slug
	// "acacias").
	BaseDomain string
	// HeaderName sobreescribe el header consultado para slug explicito.
	// Default: HeaderTenantSlug ("X-Tenant-Slug").
	HeaderName string
	// Skip permite saltar la resolucion para rutas que no requieren
	// tenant (health, superadmin, etc.).
	Skip func(r *http.Request) bool
	// Logger se usa para registrar errores internos (lookup fallido,
	// configuracion invalida). Si es nil se silencia.
	Logger *slog.Logger
}

// TenantResolver es un middleware chi-compatible que extrae el slug del
// tenant a partir del header configurado o, en su defecto, del
// subdominio del Host, lo resuelve via Registry y lo inyecta en el
// contexto del request mediante tenantctx.WithTenant.
//
// Errores emitidos (Problem+JSON via apperrors.Write):
//   - 400 Bad Request: slug ausente o con formato invalido.
//   - 404 Not Found:   slug bien formado pero desconocido por el Registry.
//   - 500 Internal:    Registry nil o error inesperado del lookup.
func TenantResolver(cfg TenantResolverConfig) func(http.Handler) http.Handler {
	headerName := strings.TrimSpace(cfg.HeaderName)
	if headerName == "" {
		headerName = HeaderTenantSlug
	}
	baseDomain := strings.ToLower(strings.TrimSpace(cfg.BaseDomain))
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

			slug, problem := resolveTenantSlug(r, headerName, baseDomain)
			if problem != nil {
				apperrors.Write(w, problem.WithInstance(r.URL.Path))
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

// resolveTenantSlug calcula el slug del tenant para el request. Da
// prioridad al header explicito (clientes movil/internos) y cae al
// subdominio del Host. Devuelve un *Problem ya formado cuando no logra
// resolver un slug valido.
func resolveTenantSlug(r *http.Request, headerName, baseDomain string) (string, *apperrors.Problem) {
	if v := strings.TrimSpace(r.Header.Get(headerName)); v != "" {
		slug := strings.ToLower(v)
		if !isValidTenantSlug(slug) {
			p := apperrors.BadRequest("tenant slug invalido")
			return "", &p
		}
		return slug, nil
	}

	slug, ok := slugFromHost(r.Host, baseDomain)
	if !ok {
		p := apperrors.BadRequest("tenant slug requerido")
		return "", &p
	}
	if !isValidTenantSlug(slug) {
		p := apperrors.BadRequest("tenant slug invalido")
		return "", &p
	}
	return slug, nil
}

// slugFromHost extrae el primer label del host cuando este es
// `<slug>.<baseDomain>`. Recorta puerto, normaliza a minusculas y
// reporta no-ok si el host coincide con baseDomain o no termina en
// `.<baseDomain>`.
func slugFromHost(host, baseDomain string) (string, bool) {
	if baseDomain == "" {
		return "", false
	}
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return "", false
	}
	if i := strings.LastIndexByte(h, ':'); i >= 0 {
		// Host sin IPv6: pgxpool no esta involucrado, basta cortar el
		// puerto. Si fuera IPv6 vendria entre corchetes y no aplicaria
		// el caso de subdominio de tenant.
		if !strings.Contains(h, "]") {
			h = h[:i]
		}
	}
	if h == baseDomain {
		return "", false
	}
	suffix := "." + baseDomain
	if !strings.HasSuffix(h, suffix) {
		return "", false
	}
	prefix := strings.TrimSuffix(h, suffix)
	if prefix == "" {
		return "", false
	}
	// El slug es el primer label antes del baseDomain. Si el cliente
	// envia `a.b.ph.localhost`, el slug logico es `a` (el resto seria
	// otro nivel de routing fuera del alcance de este middleware).
	if dot := strings.IndexByte(prefix, '.'); dot >= 0 {
		prefix = prefix[:dot]
	}
	return prefix, prefix != ""
}

// isValidTenantSlug aplica el regex y los limites de longitud al slug.
func isValidTenantSlug(slug string) bool {
	if l := len(slug); l == 0 || l > tenantSlugMaxLen {
		return false
	}
	return tenantSlugPattern.MatchString(slug)
}
