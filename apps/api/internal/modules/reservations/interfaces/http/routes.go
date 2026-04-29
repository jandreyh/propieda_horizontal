// Package http contiene los adaptadores HTTP del modulo reservations.
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

// Mount monta los endpoints del modulo reservations en r.
//
// Endpoints:
//
//	GET    /common-areas                         reservations.read
//	POST   /common-areas                         reservations.admin
//	PUT    /common-areas/{id}                    reservations.admin
//	POST   /common-areas/{id}/blackouts          reservations.admin
//	GET    /common-areas/{id}/availability       reservations.read
//	POST   /reservations                         reservations.create
//	GET    /reservations                         reservations.read
//	GET    /reservations/mine                    reservations.read
//	POST   /reservations/{id}/cancel             reservations.write
//	POST   /reservations/{id}/approve            reservations.admin
//	POST   /reservations/{id}/reject             reservations.admin
//	POST   /reservations/{id}/checkin            reservations.guard
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

	r.Route("/common-areas", func(cr chi.Router) {
		cr.With(gate("reservations.read")).Get("/", h.listCommonAreas)
		cr.With(gate("reservations.admin")).Post("/", h.createCommonArea)
		cr.With(gate("reservations.admin")).Put("/{id}", h.updateCommonArea)
		cr.With(gate("reservations.admin")).Post("/{id}/blackouts", h.createBlackout)
		cr.With(gate("reservations.read")).Get("/{id}/availability", h.getAvailability)
	})

	r.Route("/reservations", func(rv chi.Router) {
		rv.With(gate("reservations.create")).Post("/", h.createReservation)
		rv.With(gate("reservations.read")).Get("/", h.listReservations)
		rv.With(gate("reservations.read")).Get("/mine", h.listMyReservations)
		rv.With(gate("reservations.write")).Post("/{id}/cancel", h.cancelReservation)
		rv.With(gate("reservations.admin")).Post("/{id}/approve", h.approveReservation)
		rv.With(gate("reservations.admin")).Post("/{id}/reject", h.rejectReservation)
		rv.With(gate("reservations.guard")).Post("/{id}/checkin", h.checkinReservation)
	})
}
