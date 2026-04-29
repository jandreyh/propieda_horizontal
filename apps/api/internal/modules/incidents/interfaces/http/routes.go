// Package http contiene los adaptadores HTTP del modulo incidents.
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

// Mount monta los endpoints del modulo incidents en r.
//
// Endpoints:
//
//	POST   /incidents                          incidents.write
//	GET    /incidents                          incidents.read
//	GET    /incidents/{id}                     incidents.read
//	POST   /incidents/{id}/assign              incidents.assign
//	POST   /incidents/{id}/start               incidents.write
//	POST   /incidents/{id}/resolve             incidents.write
//	POST   /incidents/{id}/close               incidents.write
//	POST   /incidents/{id}/cancel              incidents.write
//	POST   /incidents/{id}/attachments         incidents.write
//	GET    /incidents/{id}/history             incidents.read
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

	r.Route("/incidents", func(ir chi.Router) {
		ir.With(gate("incidents.write")).Post("/", h.reportIncident)
		ir.With(gate("incidents.read")).Get("/", h.listIncidents)
		ir.With(gate("incidents.read")).Get("/{id}", h.getIncident)
		ir.With(gate("incidents.assign")).Post("/{id}/assign", h.assignIncident)
		ir.With(gate("incidents.write")).Post("/{id}/start", h.startIncident)
		ir.With(gate("incidents.write")).Post("/{id}/resolve", h.resolveIncident)
		ir.With(gate("incidents.write")).Post("/{id}/close", h.closeIncident)
		ir.With(gate("incidents.write")).Post("/{id}/cancel", h.cancelIncident)
		ir.With(gate("incidents.write")).Post("/{id}/attachments", h.addAttachment)
		ir.With(gate("incidents.read")).Get("/{id}/history", h.getStatusHistory)
	})
}
