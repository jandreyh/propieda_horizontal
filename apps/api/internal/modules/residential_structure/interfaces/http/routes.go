package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountOption configura el montaje del modulo. Permite inyectar guards
// (RBAC) sin que el modulo importe paquetes de autorizacion.
type MountOption func(*mountConfig)

type mountConfig struct {
	// guard mapea un namespace de permiso (ej. "structures.read") a un
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

// Mount monta los endpoints del modulo residential_structure en r.
//
// Endpoints:
//
//	POST   /structures        structures.write
//	GET    /structures        structures.read
//	GET    /structures/{id}   structures.read
//	PUT    /structures/{id}   structures.write
//	DELETE /structures/{id}   structures.write
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

	r.Route("/structures", func(sr chi.Router) {
		sr.With(gate("structures.read")).Get("/", h.listStructures)
		sr.With(gate("structures.write")).Post("/", h.createStructure)
		sr.With(gate("structures.read")).Get("/{id}", h.getStructure)
		sr.With(gate("structures.write")).Put("/{id}", h.updateStructure)
		sr.With(gate("structures.write")).Delete("/{id}", h.deleteStructure)
	})
}
