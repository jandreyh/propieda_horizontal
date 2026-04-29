// Package http contiene los adaptadores HTTP del modulo pqrs.
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

// Mount monta los endpoints del modulo pqrs en r.
//
// Endpoints:
//
//	GET    /pqrs/categories                pqrs.category.read
//	POST   /pqrs/categories                pqrs.category.write
//	PATCH  /pqrs/categories/{id}           pqrs.category.write
//	POST   /pqrs                           pqrs.ticket.create
//	GET    /pqrs                           pqrs.ticket.read
//	GET    /pqrs/{id}                      pqrs.ticket.read
//	POST   /pqrs/{id}/assign              pqrs.ticket.assign
//	POST   /pqrs/{id}/start-study         pqrs.ticket.manage
//	POST   /pqrs/{id}/respond             pqrs.ticket.respond
//	POST   /pqrs/{id}/notes               pqrs.ticket.note
//	POST   /pqrs/{id}/close               pqrs.ticket.close
//	POST   /pqrs/{id}/escalate            pqrs.ticket.manage
//	POST   /pqrs/{id}/cancel              pqrs.ticket.manage
//	GET    /pqrs/{id}/history             pqrs.ticket.read
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

	r.Route("/pqrs", func(pr chi.Router) {
		// Categories
		pr.Route("/categories", func(cr chi.Router) {
			cr.With(gate("pqrs.category.read")).Get("/", h.listCategories)
			cr.With(gate("pqrs.category.write")).Post("/", h.createCategory)
			cr.With(gate("pqrs.category.write")).Patch("/{id}", h.updateCategory)
		})

		// Tickets
		pr.With(gate("pqrs.ticket.create")).Post("/", h.fileTicket)
		pr.With(gate("pqrs.ticket.read")).Get("/", h.listTickets)
		pr.With(gate("pqrs.ticket.read")).Get("/{id}", h.getTicket)
		pr.With(gate("pqrs.ticket.assign")).Post("/{id}/assign", h.assignTicket)
		pr.With(gate("pqrs.ticket.manage")).Post("/{id}/start-study", h.startStudy)
		pr.With(gate("pqrs.ticket.respond")).Post("/{id}/respond", h.respondTicket)
		pr.With(gate("pqrs.ticket.note")).Post("/{id}/notes", h.addNote)
		pr.With(gate("pqrs.ticket.close")).Post("/{id}/close", h.closeTicket)
		pr.With(gate("pqrs.ticket.manage")).Post("/{id}/escalate", h.escalateTicket)
		pr.With(gate("pqrs.ticket.manage")).Post("/{id}/cancel", h.cancelTicket)
		pr.With(gate("pqrs.ticket.read")).Get("/{id}/history", h.getTicketHistory)
	})
}
