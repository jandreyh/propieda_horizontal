// Package tenantctx encapsula la inyeccion y extraccion del Tenant
// resuelto por el middleware TenantResolver.
//
// Reglas:
//   - Cada request operativo (todo lo que NO sea /health o /superadmin/*)
//     debe pasar por TenantResolver, que coloca un *Tenant aqui.
//   - Los handlers acceden al pool del Tenant DB unicamente via FromCtx.
//   - Nunca se persiste tenant_id en tablas operativas (CLAUDE.md), pero
//     SI viaja en el contexto del request y en logs/traces.
package tenantctx

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Tenant representa la metadata minima de un tenant resuelto, suficiente
// para que los handlers operen sin volver a consultar el Control Plane.
type Tenant struct {
	// ID es el UUID del tenant en la tabla `tenants` del Control Plane.
	ID string
	// Slug es el subdominio (ej. "acacias" en `acacias.dominio.com`).
	Slug string
	// DisplayName es el nombre humano del conjunto.
	DisplayName string
	// Pool es el pool pgx ya conectado a la base de datos del tenant.
	Pool *pgxpool.Pool
}

type ctxKey struct{}

// ErrNoTenant se devuelve cuando un handler espera tenant pero el
// contexto no lo tiene. Esto indica un error de cableado del middleware.
var ErrNoTenant = errors.New("tenantctx: no tenant in context (missing TenantResolver middleware?)")

// WithTenant retorna un contexto hijo con el tenant inyectado.
func WithTenant(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

// FromCtx extrae el tenant del contexto. Devuelve ErrNoTenant si no hay.
func FromCtx(ctx context.Context) (*Tenant, error) {
	t, ok := ctx.Value(ctxKey{}).(*Tenant)
	if !ok || t == nil {
		return nil, ErrNoTenant
	}
	return t, nil
}

// MustFromCtx extrae el tenant o entra en panico (uso permitido solo en
// tests o en codigo donde el caller ya valido la presencia).
func MustFromCtx(ctx context.Context) *Tenant {
	t, err := FromCtx(ctx)
	if err != nil {
		panic(err)
	}
	return t
}
