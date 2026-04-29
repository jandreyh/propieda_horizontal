// Package http contiene los adaptadores HTTP del modulo finance.
//
// Los handlers traducen request/response al usecase correspondiente y
// emiten errores RFC 7807 via apperrors. NO contienen logica de negocio.
package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountOption configura el montaje del modulo.
type MountOption func(*mountConfig)

type mountConfig struct {
	guard func(ns string) func(http.Handler) http.Handler
}

// WithGuard permite que el orquestador (cmd/api) pase un constructor de
// middleware RBAC que el modulo aplica por endpoint.
func WithGuard(g func(ns string) func(http.Handler) http.Handler) MountOption {
	return func(c *mountConfig) { c.guard = g }
}

// Mount monta los endpoints del modulo finance en r.
//
// Endpoints:
//
//	POST   /chart-of-accounts                         finance.write
//	GET    /chart-of-accounts                         finance.read
//	POST   /cost-centers                              finance.write
//	GET    /cost-centers                              finance.read
//	POST   /charges                                   finance.write
//	GET    /charges                                   finance.read
//	POST   /payments                                  finance.write
//	GET    /payments                                  finance.read
//	POST   /payments/{id}/allocate                    finance.write
//	POST   /payments/{id}/reverse                     finance.write
//	POST   /payments/{id}/reverse/{rid}/approve       finance.admin
//	POST   /payments/webhook/{gateway}                 (unauthenticated)
//	POST   /periods/{year}/{month}/close-soft          finance.admin
//	GET    /billing-accounts/{id}/statement            finance.read
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

	r.Route("/chart-of-accounts", func(cr chi.Router) {
		cr.With(gate("finance.write")).Post("/", h.createAccount)
		cr.With(gate("finance.read")).Get("/", h.listAccounts)
	})

	r.Route("/cost-centers", func(cr chi.Router) {
		cr.With(gate("finance.write")).Post("/", h.createCostCenter)
		cr.With(gate("finance.read")).Get("/", h.listCostCenters)
	})

	r.Route("/charges", func(cr chi.Router) {
		cr.With(gate("finance.write")).Post("/", h.createCharge)
		cr.With(gate("finance.read")).Get("/", h.listCharges)
	})

	r.Route("/payments", func(pr chi.Router) {
		pr.With(gate("finance.write")).Post("/", h.createPayment)
		pr.With(gate("finance.read")).Get("/", h.listPayments)
		pr.With(gate("finance.write")).Post("/{id}/allocate", h.allocatePayment)
		pr.With(gate("finance.write")).Post("/{id}/reverse", h.requestReversal)
		pr.With(gate("finance.admin")).Post("/{id}/reverse/{rid}/approve", h.approveReversal)
		pr.Post("/webhook/{gateway}", h.processWebhook)
	})

	r.Route("/periods", func(pr chi.Router) {
		pr.With(gate("finance.admin")).Post("/{year}/{month}/close-soft", h.closePeriodSoft)
	})

	r.Route("/billing-accounts", func(br chi.Router) {
		br.With(gate("finance.read")).Get("/{id}/statement", h.getStatement)
	})
}
