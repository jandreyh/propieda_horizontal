package middleware

import (
	"context"
	"net/http"
	"strings"

	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

// PreAuthSessionMarker es la marca que el modulo platform_identity coloca
// en `sid` para los JWT de la primera fase de login (MFA pendiente). Los
// middlewares que requieren un access token completo deben rechazar este
// marker.
const PreAuthSessionMarker = "pre-auth"

// platformAuthCtxKey es la clave bajo la cual PlatformAuth coloca las
// claims validadas. Privada para evitar colisiones — los consumidores
// usan `PlatformAuthFromCtx`.
type platformAuthCtxKey struct{}

// PlatformAuthConfig agrupa las dependencias de PlatformAuth.
type PlatformAuthConfig struct {
	// Signer es obligatorio. Verifica la firma del JWT.
	Signer *jwtsign.Signer
	// Skip permite saltar la validacion para rutas publicas (login,
	// refresh, mfa/verify, health).
	Skip func(r *http.Request) bool
}

// PlatformAuth es un middleware chi-compatible que valida el header
// Authorization Bearer, verifica el JWT con el Signer y coloca las
// claims en el contexto. Rechaza pre-auth tokens (esos solo sirven para
// /auth/mfa/verify y se procesan inline en el handler de mfa).
//
// Errores (Problem+JSON):
//   - 401 si no hay token, esta malformado, expirado, o es pre-auth.
//   - 500 si Signer es nil (mis-config).
func PlatformAuth(cfg PlatformAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}
			if cfg.Signer == nil {
				apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
				return
			}
			tok := bearerToken(r)
			if tok == "" {
				apperrors.Write(w, apperrors.Unauthorized("missing bearer token").WithInstance(r.URL.Path))
				return
			}
			claims, err := cfg.Signer.Verify(tok)
			if err != nil {
				apperrors.Write(w, apperrors.Unauthorized("invalid or expired token").WithInstance(r.URL.Path))
				return
			}
			if claims.SessionID == PreAuthSessionMarker {
				apperrors.Write(w, apperrors.Unauthorized("pre-auth token cannot access this endpoint").WithInstance(r.URL.Path))
				return
			}
			ctx := context.WithValue(r.Context(), platformAuthCtxKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PlatformAuthFromCtx extrae las claims validadas por PlatformAuth.
// Devuelve (nil, false) si el middleware no corrio para este request.
func PlatformAuthFromCtx(ctx context.Context) (*jwtsign.SessionClaims, bool) {
	c, ok := ctx.Value(platformAuthCtxKey{}).(*jwtsign.SessionClaims)
	return c, ok && c != nil
}

func bearerToken(r *http.Request) string {
	v := strings.TrimSpace(r.Header.Get("Authorization"))
	if v == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(v) <= len(prefix) || !strings.EqualFold(v[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(v[len(prefix):])
}
