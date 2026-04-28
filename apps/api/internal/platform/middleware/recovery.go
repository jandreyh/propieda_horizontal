package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// recoveryWriter envuelve un http.ResponseWriter para registrar si ya se
// escribieron headers. Permite al middleware Recovery decidir si todavia es
// seguro emitir una respuesta de error tras un panic.
type recoveryWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

// WriteHeader marca que los headers ya fueron emitidos antes de delegar al
// ResponseWriter subyacente.
func (rw *recoveryWriter) WriteHeader(status int) {
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(status)
}

// Write asegura que cualquier escritura implicita marque wroteHeader, ya
// que net/http invoca WriteHeader(200) en el primer Write si nadie lo hizo.
func (rw *recoveryWriter) Write(b []byte) (int, error) {
	rw.wroteHeader = true
	return rw.ResponseWriter.Write(b)
}

// Recovery devuelve un middleware chi-compatible que atrapa panics del
// handler downstream y los traduce a respuestas RFC 7807.
//
// Comportamiento:
//   - Si el handler panica con un valor que envuelve apperrors.Problem,
//     ese Problem se respeta tal cual (status, type, title, etc.).
//   - Cualquier otro panic se mapea a apperrors.Internal("") con
//     instance = r.URL.Path.
//   - Solo se intenta escribir la respuesta cuando los headers no se han
//     enviado todavia; en caso contrario solo se loguea.
//   - http.ErrAbortHandler se re-eleva sin tocar la respuesta para
//     respetar el contrato de net/http.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &recoveryWriter{ResponseWriter: w}

			// El defer captura el contexto del request via closure
			// (lo usamos para RequestIDFrom mas abajo).
			//nolint:contextcheck // closure recibe ctx via captura.
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				// http.ErrAbortHandler tiene semantica especial: el
				// servidor lo usa para abortar el handler sin loguear.
				// Debemos re-elevar para preservar ese contrato.
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				stack := debug.Stack()
				panicStr := fmt.Sprintf("%v", rec)

				if logger != nil {
					logger.Error(
						"panic recovered in HTTP handler",
						slog.String("request_id", RequestIDFrom(r.Context())),
						slog.String("panic", panicStr),
						slog.String("stack", string(stack)),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)
				}

				if rw.wroteHeader {
					// Ya se envio status/headers al cliente: no podemos
					// reescribir. Solo dejamos la traza en el log.
					return
				}

				problem := problemFromPanic(rec, r.URL.Path)
				apperrors.Write(rw, problem)
			}()

			next.ServeHTTP(rw, r)
		})
	}
}

// problemFromPanic resuelve el Problem RFC 7807 a emitir a partir del valor
// recuperado. Si el panic transporta un apperrors.Problem (directo o
// envuelto en un error encadenado) se respeta; en caso contrario se devuelve
// un Internal generico anclado al path del request.
func problemFromPanic(rec any, instance string) apperrors.Problem {
	if p, ok := rec.(apperrors.Problem); ok {
		if p.Instance == "" {
			p = p.WithInstance(instance)
		}
		return p
	}
	if err, ok := rec.(error); ok {
		var p apperrors.Problem
		if errors.As(err, &p) {
			if p.Instance == "" {
				p = p.WithInstance(instance)
			}
			return p
		}
	}
	return apperrors.Internal("").WithInstance(instance)
}
