package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// envLogHealthchecks es el nombre de la variable de entorno que activa el
// logging de los endpoints de salud (`/health`, `/ready`). Por defecto se
// silencian para evitar ruido en observabilidad.
const envLogHealthchecks = "LOG_HEALTHCHECKS"

// NewLogger construye un *slog.Logger configurado segun `format` y `level`.
//
// Valores aceptados:
//   - format: "json" (default) emite JSON estructurado; "text" emite logfmt.
//   - level: "debug", "info" (default), "warn", "error". Cualquier otro
//     valor cae a info para no bloquear el arranque.
//
// El destino siempre es os.Stdout para integrarse con stacks de logs
// orientados a contenedores (12-factor). Es punto de extension natural
// para enchufar OpenTelemetry / OTLP en una iteracion futura sin tocar a
// los callers.
func NewLogger(format string, level string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// parseLevel mapea el string de nivel a slog.Level. Defaults a info si el
// valor es desconocido.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// responseRecorder envuelve un http.ResponseWriter para capturar el status
// code y la cantidad de bytes escritos al cuerpo.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

// newResponseRecorder construye un recorder con status 200 por defecto
// (alineado con net/http: si nadie llama WriteHeader, la respuesta es 200).
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, status: http.StatusOK}
}

// WriteHeader captura el status code la primera vez que se invoca y
// delega en el ResponseWriter subyacente.
func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

// Write delega en el ResponseWriter subyacente y acumula los bytes
// escritos. Si nadie llamo WriteHeader, marca la respuesta como 200.
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err //nolint:wrapcheck // delegacion directa al writer subyacente.
}

// Logging es un middleware chi-compatible que emite UN log estructurado
// por request, con campos de correlacion (request_id, tenant) y metricas
// basicas (status, latencia, bytes). El nivel se eleva automaticamente
// segun el status (>=500 error, >=400 warn, resto info).
//
// Punto de extension: los campos emitidos aqui son los mismos que se
// propagaran como atributos de span cuando se integre OpenTelemetry.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	logHealth := strings.EqualFold(strings.TrimSpace(os.Getenv(envLogHealthchecks)), "true")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !logHealth && isHealthPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rec := newResponseRecorder(w)

			next.ServeHTTP(rec, r)

			duration := time.Since(start)

			attrs := make([]slog.Attr, 0, 12)
			if id := RequestIDFrom(r.Context()); id != "" {
				attrs = append(attrs, slog.String("request_id", id))
			}
			if t, err := tenantctx.FromCtx(r.Context()); err == nil && t != nil {
				attrs = append(attrs,
					slog.String("tenant_id", t.ID),
					slog.String("tenant_slug", t.Slug),
				)
			}
			attrs = append(attrs,
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
				slog.String("client_ip", clientIP(r)),
				slog.String("user_agent", r.UserAgent()),
			)

			logger.LogAttrs(r.Context(), levelForStatus(rec.status), "http_request", attrs...)
		})
	}
}

// isHealthPath reporta si el path corresponde a un endpoint de salud que
// se silencia por defecto.
func isHealthPath(p string) bool {
	return p == "/health" || p == "/ready"
}

// levelForStatus mapea el status HTTP al nivel de log apropiado.
func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
