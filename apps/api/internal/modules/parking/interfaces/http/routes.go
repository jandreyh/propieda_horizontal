// Package http contiene los adaptadores HTTP del modulo parking.
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

// Mount monta los endpoints del modulo parking en r.
//
// Endpoints:
//
//	POST   /parking-spaces                            parking.write
//	GET    /parking-spaces                            parking.read
//	PUT    /parking-spaces/{id}                       parking.write
//	POST   /parking-spaces/{id}/assign                parking.assign
//	POST   /parking-assignments/{id}/release          parking.assign
//	GET    /units/{id}/parking                        parking.read
//	POST   /parking-visitor-reservations              parking.visitor.create
//	POST   /parking-visitor-reservations/{id}/cancel  parking.write
//	GET    /parking-visitor-reservations               parking.read
//	POST   /parking-lotteries/run                     parking.lottery.run
//	GET    /parking-lotteries/{id}/results             parking.read
//	GET    /guard/parking/today                       parking.guard.read
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

	r.Route("/parking-spaces", func(pr chi.Router) {
		pr.With(gate("parking.write")).Post("/", h.createSpace)
		pr.With(gate("parking.read")).Get("/", h.listSpaces)
		pr.With(gate("parking.write")).Put("/{id}", h.updateSpace)
		pr.With(gate("parking.assign")).Post("/{id}/assign", h.assignSpace)
	})

	r.Route("/parking-assignments", func(ar chi.Router) {
		ar.With(gate("parking.assign")).Post("/{id}/release", h.releaseAssignment)
	})

	// Endpoint anidado bajo units — no usamos Route("/units") porque ese
	// prefijo ya fue registrado por el modulo units. Usamos Get directo.
	r.With(gate("parking.read")).Get("/units/{id}/parking", h.getUnitParking)

	r.Route("/parking-visitor-reservations", func(vr chi.Router) {
		vr.With(gate("parking.visitor.create")).Post("/", h.createVisitorReservation)
		vr.With(gate("parking.read")).Get("/", h.listVisitorReservations)
		vr.With(gate("parking.write")).Post("/{id}/cancel", h.cancelVisitorReservation)
	})

	r.Route("/parking-lotteries", func(lr chi.Router) {
		lr.With(gate("parking.lottery.run")).Post("/run", h.runLottery)
		lr.With(gate("parking.read")).Get("/{id}/results", h.getLotteryResults)
	})

	r.Route("/guard/parking", func(gr chi.Router) {
		gr.With(gate("parking.guard.read")).Get("/today", h.guardParkingToday)
	})
}
