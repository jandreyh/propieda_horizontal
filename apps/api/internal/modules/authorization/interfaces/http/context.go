// Package http implementa la interfaz HTTP del modulo authorization
// (handlers, middleware, rutas chi).
package http

import "context"

// userIDKey es la key privada usada para inyectar el user_id en el
// contexto del request. Un middleware de autenticacion previo (fuera de
// este modulo) coloca el user_id aqui via WithUserID.
type userIDKey struct{}

// sessionIDKey es la key privada usada para inyectar el session_id (de
// authn). Sirve como key de cache de permisos efectivos.
type sessionIDKey struct{}

// WithUserID retorna un contexto hijo con el user_id inyectado.
func WithUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromCtx extrae el user_id del contexto. Devuelve "" si no hay.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey{}).(string)
	return v
}

// WithSessionID retorna un contexto hijo con el session_id inyectado.
func WithSessionID(ctx context.Context, sid string) context.Context {
	if sid == "" {
		return ctx
	}
	return context.WithValue(ctx, sessionIDKey{}, sid)
}

// SessionIDFromCtx extrae el session_id del contexto.
func SessionIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(sessionIDKey{}).(string)
	return v
}
