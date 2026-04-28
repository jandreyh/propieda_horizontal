package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountOption configura el montaje del modulo. Usado para inyectar guards
// (RBAC) sin que el modulo importe paquetes de autorizacion.
type MountOption func(*mountConfig)

type mountConfig struct {
	// guard mapea un namespace de permiso (ej. "settings.read") a un
	// middleware chi que aplica el chequeo. Si guard es nil, no se aplica
	// gate y el handler es accesible (util en tests y desarrollo local).
	guard func(ns string) func(http.Handler) http.Handler
}

// WithGuard permite que el orquestador (cmd/api) pase un constructor de
// middleware RBAC que el modulo aplica por endpoint. La firma evita que
// este paquete importe el modulo authorization.
func WithGuard(g func(ns string) func(http.Handler) http.Handler) MountOption {
	return func(c *mountConfig) { c.guard = g }
}

// Mount monta los endpoints del modulo tenant_config en r.
//
// Endpoints (segun spec del modulo):
//
//	GET    /settings           settings.read
//	GET    /settings/{key}     settings.read
//	PUT    /settings/{key}     settings.write
//	DELETE /settings/{key}     settings.write
//	GET    /branding           branding.read
//	PUT    /branding           branding.write
//
// El modulo NO conoce el modulo authorization: si se desea gating RBAC,
// se inyecta via WithGuard. Si no, el guard es no-op.
func Mount(r chi.Router, deps Dependencies, opts ...MountOption) {
	cfg := &mountConfig{}
	for _, o := range opts {
		o(cfg)
	}

	h := newHandlers(deps)

	gate := func(ns string) func(http.Handler) http.Handler {
		if cfg.guard == nil {
			return func(next http.Handler) http.Handler { return next }
		}
		return cfg.guard(ns)
	}

	r.Route("/settings", func(sr chi.Router) {
		sr.With(gate("settings.read")).Get("/", h.listSettings)
		sr.With(gate("settings.read")).Get("/{key}", h.getSetting)
		sr.With(gate("settings.write")).Put("/{key}", h.putSetting)
		sr.With(gate("settings.write")).Delete("/{key}", h.deleteSetting)
	})

	r.Route("/branding", func(br chi.Router) {
		br.With(gate("branding.read")).Get("/", h.getBranding)
		br.With(gate("branding.write")).Put("/", h.putBranding)
	})
}
