// Package http contiene los adaptadores HTTP del modulo notifications.
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

// Mount monta los endpoints del modulo notifications en r.
//
// Endpoints:
//
//	GET    /notifications/preferences                    notifications.read
//	PATCH  /notifications/preferences                    notifications.write
//	POST   /notifications/consents                       notifications.write
//	DELETE /notifications/consents/{channel}              notifications.write
//	POST   /notifications/push-tokens                    notifications.write
//	DELETE /notifications/push-tokens/{id}                notifications.write
//	GET    /notifications/templates                      notifications.admin
//	POST   /notifications/templates                      notifications.admin
//	PATCH  /notifications/templates/{id}                  notifications.admin
//	GET    /notifications/provider-configs               notifications.admin
//	PATCH  /notifications/provider-configs               notifications.admin
//	POST   /notifications/broadcast                      notifications.broadcast
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

	r.Route("/notifications", func(nr chi.Router) {
		// Preferences (user-facing).
		nr.With(gate("notifications.read")).Get("/preferences", h.listPreferences)
		nr.With(gate("notifications.write")).Patch("/preferences", h.patchPreferences)

		// Consents (user-facing).
		nr.With(gate("notifications.write")).Post("/consents", h.createConsent)
		nr.With(gate("notifications.write")).Delete("/consents/{channel}", h.revokeConsent)

		// Push tokens (user-facing).
		nr.With(gate("notifications.write")).Post("/push-tokens", h.createPushToken)
		nr.With(gate("notifications.write")).Delete("/push-tokens/{id}", h.deletePushToken)

		// Templates (admin).
		nr.With(gate("notifications.admin")).Get("/templates", h.listTemplates)
		nr.With(gate("notifications.admin")).Post("/templates", h.createTemplate)
		nr.With(gate("notifications.admin")).Patch("/templates/{id}", h.updateTemplate)

		// Provider configs (admin).
		nr.With(gate("notifications.admin")).Get("/provider-configs", h.listProviderConfigs)
		nr.With(gate("notifications.admin")).Patch("/provider-configs", h.updateProviderConfig)

		// Broadcast (admin + MFA).
		nr.With(gate("notifications.broadcast")).Post("/broadcast", h.broadcast)
	})
}
