// Package middleware contiene middlewares HTTP transversales para el
// servidor chi. Son agnosticos del dominio y no dependen de modulos de
// negocio.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
)

// HeaderRequestID es el nombre canonico del header HTTP que transporta el
// identificador de correlacion del request entre cliente, servidor, logs y
// traces.
const HeaderRequestID = "X-Request-ID"

// requestIDPattern define el formato aceptado para un id propuesto por el
// cliente: 8 a 128 caracteres alfanumericos mas `.`, `_` y `-`.
var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,128}$`)

// ctxKey es un tipo no exportado para evitar colisiones de claves en el
// context.Context.
type ctxKey struct{}

// requestIDKey es la clave concreta usada para almacenar el request id.
var requestIDKey = ctxKey{}

// WithRequestID retorna un contexto hijo con el request id inyectado.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFrom extrae el request id del contexto. Devuelve cadena vacia
// cuando no hay id presente.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestID es un middleware chi-compatible que asegura que cada request
// tenga un identificador de correlacion estable.
//
// Comportamiento:
//   - Si el cliente envia X-Request-ID y cumple el formato aceptado, se
//     reutiliza tal cual.
//   - En caso contrario (vacio, demasiado corto, caracteres invalidos) se
//     genera un UUIDv4 nuevo via crypto/rand.
//   - El id se inyecta en el contexto del request y se escribe en el
//     response header X-Request-ID antes de invocar al siguiente handler.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if !isValidRequestID(id) {
			id = newUUIDv4()
		}

		w.Header().Set(HeaderRequestID, id)
		ctx := WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValidRequestID valida el id propuesto contra el regex permitido.
func isValidRequestID(id string) bool {
	if id == "" {
		return false
	}
	return requestIDPattern.MatchString(id)
}

// newUUIDv4 genera un UUID version 4 (RFC 4122) usando crypto/rand.
// Ante un fallo improbable de la fuente de entropia, retorna un id con
// prefijo determinista para no romper la cadena de logs.
func newUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback agnostico: usa el error para producir un id valido
		// segun el regex aceptado, evitando panic en el hot path HTTP.
		return fmt.Sprintf("reqid-fallback-%x", err.Error())
	}
	// Version 4 (0100 en los 4 bits altos del byte 6).
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant RFC 4122 (10xx en los 2 bits altos del byte 8).
	b[8] = (b[8] & 0x3f) | 0x80

	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])
	return string(out[:])
}
