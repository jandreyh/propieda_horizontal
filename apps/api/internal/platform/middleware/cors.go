package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig controla la configuracion del middleware CORS.
//
// AllowedOrigins es una lista exacta de origenes permitidos. Si se incluye
// el comodin "*" se aceptara cualquier origen (uso solo en dev).
type CORSConfig struct {
	AllowedOrigins []string
}

// CORS devuelve un middleware chi-compatible que aplica CORS para los
// frontends del proyecto (Next.js en :3000, Flutter web en :3002).
//
// Para clientes que mandan credentials (no es nuestro caso porque el
// JWT viaja en Authorization header) NO se devuelve "*" como origin
// reflectido sino el origin exacto del request.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowAny := false
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAny = true
			continue
		}
		allowed[strings.ToLower(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				normalized := strings.ToLower(origin)
				if _, ok := allowed[normalized]; ok || allowAny {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set(
						"Access-Control-Allow-Headers",
						"Authorization, Content-Type, X-Tenant-Slug, X-Request-Id",
					)
					w.Header().Set(
						"Access-Control-Allow-Methods",
						"GET, POST, PUT, PATCH, DELETE, OPTIONS",
					)
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
