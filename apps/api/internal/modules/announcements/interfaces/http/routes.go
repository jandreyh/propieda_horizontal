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

// Mount monta los endpoints del modulo announcements en r.
//
// Endpoints:
//
//	POST   /announcements              announcement.publish
//	GET    /announcements/feed         announcement.read
//	GET    /announcements/{id}         announcement.read
//	POST   /announcements/{id}/ack     announcement.read
//	DELETE /announcements/{id}         announcement.delete
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

	r.Route("/announcements", func(ar chi.Router) {
		ar.With(gate("announcement.publish")).Post("/", h.createAnnouncement)
		ar.With(gate("announcement.read")).Get("/feed", h.listFeed)
		ar.With(gate("announcement.read")).Get("/{id}", h.getAnnouncement)
		ar.With(gate("announcement.read")).Post("/{id}/ack", h.ackAnnouncement)
		ar.With(gate("announcement.delete")).Delete("/{id}", h.archiveAnnouncement)
	})
}
