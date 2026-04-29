// Package http contiene los adaptadores HTTP del modulo assemblies.
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

// Mount monta los endpoints del modulo assemblies en r.
//
// Endpoints:
//
//	POST   /assemblies                           assemblies.write
//	GET    /assemblies                           assemblies.read
//	GET    /assemblies/{id}                      assemblies.read
//	POST   /assemblies/{id}/call                 assemblies.write
//	POST   /assemblies/{id}/start                assemblies.manage
//	POST   /assemblies/{id}/close                assemblies.manage
//	POST   /assemblies/{id}/attendances          assemblies.attendance
//	POST   /assemblies/{id}/proxies              assemblies.proxy
//	POST   /assemblies/{id}/motions              assemblies.write
//	POST   /motions/{id}/open-voting             assemblies.manage
//	POST   /motions/{id}/close-voting            assemblies.manage
//	POST   /motions/{id}/votes                   assemblies.vote
//	GET    /motions/{id}/results                 assemblies.read
//	POST   /assemblies/{id}/act                  assemblies.act.write
//	POST   /acts/{id}/sign                       assemblies.act.sign
//	GET    /acts/{id}                            assemblies.read
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

	r.Route("/assemblies", func(ar chi.Router) {
		ar.With(gate("assemblies.write")).Post("/", h.createAssembly)
		ar.With(gate("assemblies.read")).Get("/", h.listAssemblies)
		ar.With(gate("assemblies.read")).Get("/{id}", h.getAssembly)
		ar.With(gate("assemblies.write")).Post("/{id}/call", h.publishCall)
		ar.With(gate("assemblies.manage")).Post("/{id}/start", h.startAssembly)
		ar.With(gate("assemblies.manage")).Post("/{id}/close", h.closeAssembly)
		ar.With(gate("assemblies.attendance")).Post("/{id}/attendances", h.registerAttendance)
		ar.With(gate("assemblies.proxy")).Post("/{id}/proxies", h.registerProxy)
		ar.With(gate("assemblies.write")).Post("/{id}/motions", h.createMotion)
		ar.With(gate("assemblies.act.write")).Post("/{id}/act", h.createAct)
	})

	r.Route("/motions", func(mr chi.Router) {
		mr.With(gate("assemblies.manage")).Post("/{id}/open-voting", h.openVoting)
		mr.With(gate("assemblies.manage")).Post("/{id}/close-voting", h.closeVoting)
		mr.With(gate("assemblies.vote")).Post("/{id}/votes", h.castVote)
		mr.With(gate("assemblies.read")).Get("/{id}/results", h.getMotionResults)
	})

	r.Route("/acts", func(acr chi.Router) {
		acr.With(gate("assemblies.act.sign")).Post("/{id}/sign", h.signAct)
		acr.With(gate("assemblies.read")).Get("/{id}", h.getAct)
	})
}
