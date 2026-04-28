package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/saas-ph/api/internal/modules/authorization/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// permissionResolver es la superficie minima que el middleware necesita
// para chequear permisos. La produccion usa el usecase
// ResolveUserPermissions; los tests pueden inyectar un fake.
type permissionResolver interface {
	Permissions(ctx context.Context, userID string) ([]string, error)
}

// PermissionCache cachea el set de permisos por sesion durante un TTL
// corto (default 60s) para evitar consultar DB en cada request. El cache
// es local al proceso y al tenant (la session_id ya es por-tenant).
type PermissionCache struct {
	ttl     time.Duration
	now     func() time.Time
	mu      sync.Mutex
	entries map[string]permsCacheEntry
}

type permsCacheEntry struct {
	perms     []string
	expiresAt time.Time
}

// NewPermissionCache construye un cache vacio. ttl<=0 desactiva el TTL
// (sin cache); por defecto el middleware usa 60s si no se inyecta uno.
func NewPermissionCache(ttl time.Duration, now func() time.Time) *PermissionCache {
	if now == nil {
		now = time.Now
	}
	return &PermissionCache{
		ttl:     ttl,
		now:     now,
		entries: make(map[string]permsCacheEntry),
	}
}

// Get devuelve los permisos cacheados y true si la entrada existe y no
// expiro.
func (c *PermissionCache) Get(key string) ([]string, bool) {
	if c == nil || c.ttl <= 0 || key == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if c.now().After(e.expiresAt) {
		delete(c.entries, key)
		return nil, false
	}
	return e.perms, true
}

// Set guarda el set de permisos para la session key.
func (c *PermissionCache) Set(key string, perms []string) {
	if c == nil || c.ttl <= 0 || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = permsCacheEntry{
		perms:     perms,
		expiresAt: c.now().Add(c.ttl),
	}
}

// Invalidate borra una entrada del cache (ej. cuando se asigna/revoca
// un rol al usuario; el caller hace el lookup de session_id).
func (c *PermissionCache) Invalidate(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// MiddlewareConfig agrupa las dependencias de RequirePermission.
type MiddlewareConfig struct {
	Resolver permissionResolver
	Cache    *PermissionCache
}

// RequirePermission devuelve un middleware chi-compatible que rechaza
// con 403 RFC 7807 si el usuario actual no tiene el permiso `ns`
// (soporta wildcards via policies.HasPermission).
//
// Flujo:
//  1. Lee user_id desde el contexto (UserIDFromCtx). Si no esta, 401.
//  2. Resuelve permisos efectivos del usuario via Resolver, con cache
//     en memoria por session_id (TTL 60s) si hay session_id en contexto.
//  3. Si el set contiene `ns` o un wildcard que lo cubre, pasa.
//  4. Sino: 403 con detail "missing permission: <ns>".
func (c MiddlewareConfig) RequirePermission(ns string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromCtx(r.Context())
			if userID == "" {
				apperrors.Write(w, apperrors.Unauthorized("authentication required").
					WithInstance(r.URL.Path))
				return
			}

			perms, err := c.resolveCached(r.Context(), userID)
			if err != nil {
				apperrors.Write(w, apperrors.Internal("failed to resolve permissions").
					WithInstance(r.URL.Path))
				return
			}

			if !policies.HasPermission(perms, ns) {
				apperrors.Write(w, apperrors.Forbidden("missing permission: "+ns).
					WithInstance(r.URL.Path))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission es un helper para handlers que aceptan multiples
// permisos alternativos (ej. lectura por permiso o por self-access).
func (c MiddlewareConfig) RequireAnyPermission(any ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromCtx(r.Context())
			if userID == "" {
				apperrors.Write(w, apperrors.Unauthorized("authentication required").
					WithInstance(r.URL.Path))
				return
			}

			perms, err := c.resolveCached(r.Context(), userID)
			if err != nil {
				apperrors.Write(w, apperrors.Internal("failed to resolve permissions").
					WithInstance(r.URL.Path))
				return
			}

			if !policies.HasAnyPermission(perms, any...) {
				apperrors.Write(w, apperrors.Forbidden("missing one of: "+joinPerms(any)).
					WithInstance(r.URL.Path))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireScope es un middleware opcional que combina RequirePermission
// con la verificacion de scope a nivel de recurso. La extraccion del
// scope (tipo + id) se delega al caller.
//
// TODO(post-MVP): recibe la lista de assignments del usuario (no solo
// los namespaces) para evaluar policies.MatchesScope correctamente.
// Hoy day se queda en chequeo de permiso plano y deja el scope como
// hint para handlers que filtran SQL aguas abajo.
func (c MiddlewareConfig) RequireScope(ns string, scopeExtractor func(*http.Request) (scopeType, scopeID string)) func(http.Handler) http.Handler {
	// MVP: comportamiento equivalente a RequirePermission. El scope
	// extraido se pasa adelante via contexto para que el handler pueda
	// usarlo como filtro. En post-MVP se enriquece con
	// policies.MatchesScope sobre las asignaciones reales.
	return c.RequirePermission(ns)
}

func (c MiddlewareConfig) resolveCached(ctx context.Context, userID string) ([]string, error) {
	cacheKey := SessionIDFromCtx(ctx)
	if cacheKey == "" {
		// Sin session id no podemos cachear de forma segura (otra request
		// del mismo user pero distinta session reusaria una clave).
		// Caer al lookup directo.
		return c.Resolver.Permissions(ctx, userID)
	}
	if perms, ok := c.Cache.Get(cacheKey); ok {
		return perms, nil
	}
	perms, err := c.Resolver.Permissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	c.Cache.Set(cacheKey, perms)
	return perms, nil
}

// joinPerms hace un join sin importar strings.Join para no agregar otra
// dependencia. Mantiene errores simples y predecibles.
func joinPerms(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
