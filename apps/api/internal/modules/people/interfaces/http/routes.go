package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MountOption configura el montaje del modulo. Usado para inyectar guards
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

// Mount monta los endpoints del modulo people en r.
//
// Endpoints:
//
//	POST   /vehicles                                   vehicles.write
//	GET    /vehicles                                   vehicles.read    (?plate= opcional)
//	GET    /vehicles/{id}                              vehicles.read
//	POST   /units/{unitID}/vehicles                    vehicles.assign
//	GET    /units/{unitID}/vehicles                    vehicles.read
//	DELETE /units/{unitID}/vehicles/{assignmentID}     vehicles.assign
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

	r.Route("/vehicles", func(vr chi.Router) {
		vr.With(gate("vehicles.write")).Post("/", h.createVehicle)
		vr.With(gate("vehicles.read")).Get("/", h.listVehicles)
		vr.With(gate("vehicles.read")).Get("/{id}", h.getVehicle)
	})

	r.Route("/units/{unitID}/vehicles", func(ur chi.Router) {
		ur.With(gate("vehicles.read")).Get("/", h.listVehiclesForUnit)
		ur.With(gate("vehicles.assign")).Post("/", h.assignVehicle)
		ur.With(gate("vehicles.assign")).Delete("/{assignmentID}", h.endAssignment)
	})
}
