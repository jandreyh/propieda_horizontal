// Package http contiene los adaptadores HTTP del modulo access_control.
//
// Los handlers traducen request/response al usecase correspondiente y
// emiten errores RFC 7807 via apperrors. NO contienen logica de negocio.
package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountOption configura el montaje del modulo. Permite inyectar guards
// (RBAC) sin que el modulo importe paquetes de autorizacion.
type MountOption func(*mountConfig)

type mountConfig struct {
	guard func(ns string) func(http.Handler) http.Handler
}

// WithGuard permite que el orquestador (cmd/api) pase un constructor de
// middleware RBAC que el modulo aplica por endpoint.
func WithGuard(g func(ns string) func(http.Handler) http.Handler) MountOption {
	return func(c *mountConfig) { c.guard = g }
}

// Mount monta los endpoints del modulo access_control en r.
//
// Endpoints:
//
//	POST   /visitor-preregistrations          visit.create     (residente)
//	POST   /visits/checkin-by-qr              visit.create     (guard)
//	POST   /visits/checkin-manual             visit.create     (guard)
//	POST   /visits/{id}/checkout              visit.create
//	GET    /visits/active                     visit.read
//	POST   /blacklist                         blacklist.write  (namespace nuevo)
//	GET    /blacklist                         blacklist.read   (namespace nuevo)
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

	r.Route("/visitor-preregistrations", func(pr chi.Router) {
		pr.With(gate("visit.create")).Post("/", h.createPreRegistration)
	})

	r.Route("/visits", func(vr chi.Router) {
		vr.With(gate("visit.create")).Post("/checkin-by-qr", h.checkinByQR)
		vr.With(gate("visit.create")).Post("/checkin-manual", h.checkinManual)
		vr.With(gate("visit.create")).Post("/{id}/checkout", h.checkout)
		vr.With(gate("visit.read")).Get("/active", h.listActive)
	})

	r.Route("/blacklist", func(br chi.Router) {
		br.With(gate("blacklist.write")).Post("/", h.createBlacklist)
		br.With(gate("blacklist.read")).Get("/", h.listBlacklist)
	})
}
