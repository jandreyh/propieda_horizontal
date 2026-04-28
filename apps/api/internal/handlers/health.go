// Package handlers contiene los handlers HTTP transversales que NO
// pertenecen a un modulo de negocio (health, ready, version).
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/tenantctx"
	"github.com/saas-ph/api/internal/version"
)

// Health responde 200 OK siempre que el proceso este vivo. NO toca DB
// para no caer si Postgres tiene un hipo. Ideal para liveness probe.
func Health(w http.ResponseWriter, _ *http.Request) {
	resp := struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Time    string `json:"time"`
	}{
		Status:  "ok",
		Version: version.Version,
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Ready hace un ping al pool del Control Plane. NO requiere tenant.
// Ideal para readiness probe del Control Plane.
func Ready(centralPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if centralPool == nil {
			apperrors.Write(w, apperrors.Internal("control plane pool no inicializado").
				WithInstance(r.URL.Path))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := centralPool.Ping(ctx); err != nil {
			apperrors.Write(w, apperrors.New(http.StatusServiceUnavailable,
				"db-unavailable", "Service Unavailable",
				"control plane database not reachable").
				WithInstance(r.URL.Path))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}

// TenantReady hace un ping al pool del Tenant DB resuelto en el contexto.
// REQUIERE pasar por el middleware TenantResolver.
func TenantReady(w http.ResponseWriter, r *http.Request) {
	t, err := tenantctx.FromCtx(r.Context())
	if err != nil {
		apperrors.Write(w, apperrors.BadRequest("tenant not resolved").
			WithInstance(r.URL.Path))
		return
	}
	if t.Pool == nil {
		apperrors.Write(w, apperrors.Internal("tenant pool nil").
			WithInstance(r.URL.Path))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := t.Pool.Ping(ctx); err != nil {
		apperrors.Write(w, apperrors.New(http.StatusServiceUnavailable,
			"db-unavailable", "Service Unavailable",
			"tenant database not reachable").
			WithInstance(r.URL.Path))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready", "tenant": t.Slug})
}
