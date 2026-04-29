// Package http contiene los adaptadores HTTP del modulo penalties.
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

// Mount monta los endpoints del modulo penalties en r.
//
// Endpoints:
//
//	GET    /penalty-catalog                        penalties.catalog.read
//	POST   /penalty-catalog                        penalties.catalog.write
//	PATCH  /penalty-catalog/{id}                   penalties.catalog.write
//	POST   /penalties                              penalties.write
//	GET    /penalties                              penalties.read
//	POST   /penalties/{id}/notify                  penalties.write
//	POST   /penalties/{id}/council-approve         penalties.write
//	POST   /penalties/{id}/confirm                 penalties.write
//	POST   /penalties/{id}/settle                  penalties.write
//	POST   /penalties/{id}/cancel                  penalties.write
//	POST   /penalties/{id}/appeals                 penalties.appeal.submit
//	POST   /penalties/{id}/appeals/{aid}/resolve   penalties.appeal.resolve
//	GET    /penalties/{id}/history                 penalties.read
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

	r.Route("/penalty-catalog", func(cr chi.Router) {
		cr.With(gate("penalties.catalog.read")).Get("/", h.listCatalog)
		cr.With(gate("penalties.catalog.write")).Post("/", h.createCatalogEntry)
		cr.With(gate("penalties.catalog.write")).Patch("/{id}", h.updateCatalogEntry)
	})

	r.Route("/penalties", func(pr chi.Router) {
		pr.With(gate("penalties.write")).Post("/", h.imposePenalty)
		pr.With(gate("penalties.read")).Get("/", h.listPenalties)

		pr.Route("/{id}", func(sr chi.Router) {
			sr.With(gate("penalties.write")).Post("/notify", h.notifyPenalty)
			sr.With(gate("penalties.write")).Post("/council-approve", h.councilApprovePenalty)
			sr.With(gate("penalties.write")).Post("/confirm", h.confirmPenalty)
			sr.With(gate("penalties.write")).Post("/settle", h.settlePenalty)
			sr.With(gate("penalties.write")).Post("/cancel", h.cancelPenalty)

			sr.With(gate("penalties.appeal.submit")).Post("/appeals", h.submitAppeal)
			sr.With(gate("penalties.appeal.resolve")).Post("/appeals/{aid}/resolve", h.resolveAppeal)

			sr.With(gate("penalties.read")).Get("/history", h.getPenaltyHistory)
		})
	})
}
