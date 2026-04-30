package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig controla los allowed origins. AllowAll permite cualquiera
// (util en dev). En produccion fijar Origins explicitos.
type CORSConfig struct {
	AllowedOrigins []string // si vacio + AllowAll=false → no se emite header
	AllowAll       bool
}

// CORS es un middleware permisivo para desarrollo: permite localhost y
// 127.0.0.1 en cualquier puerto si AllowAll=false y Origins esta vacio.
// Soporta preflight OPTIONS y emite los headers necesarios para que el
// frontend mande Authorization Bearer.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowOrigin(cfg, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers",
					"Authorization, Content-Type, X-Tenant-Slug, X-Request-ID, Idempotency-Key")
				w.Header().Set("Access-Control-Allow-Methods",
					"GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowOrigin(cfg CORSConfig, origin string) bool {
	if cfg.AllowAll {
		return true
	}
	for _, o := range cfg.AllowedOrigins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	// En dev por defecto permitir localhost + 127.0.0.1.
	low := strings.ToLower(origin)
	if strings.HasPrefix(low, "http://localhost:") ||
		strings.HasPrefix(low, "http://127.0.0.1:") ||
		strings.HasPrefix(low, "http://[::1]:") {
		return true
	}
	return false
}
