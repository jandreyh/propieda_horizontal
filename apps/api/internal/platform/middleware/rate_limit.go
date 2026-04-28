package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// defaultCleanupInterval es el TTL por defecto para evictar buckets
// inactivos cuando el caller no especifica CleanupInterval en
// RateLimitConfig.
const defaultCleanupInterval = 10 * time.Minute

// RateLimitConfig describe la politica del middleware RateLimit.
//
// Los campos son inmutables tras construir el middleware: el llamante
// debe rellenar el struct y pasarlo por valor a RateLimit.
type RateLimitConfig struct {
	// RequestsPerSecond es la tasa sostenida de recarga del bucket
	// (tokens por segundo). Debe ser > 0 para que el limitador opere.
	RequestsPerSecond float64

	// Burst es la capacidad maxima del bucket (numero de requests que
	// pueden pasar de forma instantanea antes de tener que esperar a la
	// recarga). Debe ser >= 1.
	Burst int

	// KeyFunc extrae la clave de identidad del request (tipicamente
	// IP de cliente o `tenant_id`). Si es nil se usa el IP del cliente
	// derivado de X-Forwarded-For / RemoteAddr.
	KeyFunc func(r *http.Request) string

	// CleanupInterval define cada cuanto tiempo se considera obsoleto
	// un bucket inactivo. Buckets cuyo `last < now - 2*CleanupInterval`
	// son purgados. Si es 0, se usa 10 minutos.
	CleanupInterval time.Duration

	// Now permite inyectar un reloj falso en tests. Si es nil se usa
	// time.Now.
	Now func() time.Time
}

// bucket es el estado por-clave del algoritmo token-bucket.
type bucket struct {
	tokens float64
	last   time.Time
}

// limiter agrupa el estado mutable compartido entre todas las requests:
// el mapa de buckets, el mutex que lo protege, la config inmutable y un
// contador de accesos para disparar sweeps de eviccion on-demand.
type limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	cfg      RateLimitConfig
	accesses int
}

// RateLimit construye un middleware chi-compatible que aplica un
// algoritmo token-bucket en memoria por proceso, particionado por la
// clave devuelta por cfg.KeyFunc.
//
// Comportamiento:
//   - En cada request resuelve la key, recarga tokens segun el tiempo
//     transcurrido (cap a Burst) y consume 1 token.
//   - Si no quedan tokens, responde 429 Too Many Requests con un
//     Problem RFC 7807 y header Retry-After (segundos enteros).
//   - Eviccion: para evitar goroutines de fondo no cancelables, el
//     barrido de buckets obsoletos se ejecuta on-demand cada
//     `Burst*100` accesos (o cada 100 accesos si Burst <= 0). Buckets
//     con `last < now - 2*CleanupInterval` son eliminados.
//
// La construccion no falla con configuraciones invalidas: aplica los
// defaults `CleanupInterval=10min` y `Now=time.Now`, y si `KeyFunc` es
// nil cae en el helper interno clientIP.
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = defaultCleanupInterval
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = clientIP
	}

	l := &limiter{
		buckets: make(map[string]*bucket),
		cfg:     cfg,
	}

	sweepEvery := cfg.Burst * 100
	if sweepEvery <= 0 {
		sweepEvery = 100
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.KeyFunc(r)
			now := cfg.Now()

			allowed, retryAfter := l.take(key, now, sweepEvery)
			if !allowed {
				secs := int(math.Ceil(retryAfter.Seconds()))
				if secs < 1 {
					secs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				p := apperrors.TooManyRequests("rate limit exceeded").
					WithInstance(r.URL.Path)
				apperrors.Write(w, p)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// take aplica el algoritmo token-bucket sobre el bucket asociado a key.
// Devuelve (true, 0) si la request fue admitida o (false, retryAfter)
// con la espera estimada para que se acumule 1 token cuando se rechaza.
//
// El sweep on-demand se ejecuta dentro del mismo lock para evitar
// disputar el mutex dos veces. Por construccion el sweep es O(buckets)
// pero amortizado: solo corre una vez cada sweepEvery accesos.
func (l *limiter) take(key string, now time.Time, sweepEvery int) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.accesses++
	if l.accesses >= sweepEvery {
		l.accesses = 0
		l.sweepLocked(now)
	}

	b, ok := l.buckets[key]
	if !ok {
		// Bucket nuevo: arranca lleno (Burst tokens) y consume 1.
		l.buckets[key] = &bucket{
			tokens: float64(l.cfg.Burst) - 1,
			last:   now,
		}
		return true, 0
	}

	// Recarga proporcional al tiempo transcurrido.
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.cfg.RequestsPerSecond
		if b.tokens > float64(l.cfg.Burst) {
			b.tokens = float64(l.cfg.Burst)
		}
	}
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	// No hay token entero disponible. Calcula cuanto falta para 1.
	missing := 1 - b.tokens
	var wait time.Duration
	if l.cfg.RequestsPerSecond > 0 {
		wait = time.Duration((missing / l.cfg.RequestsPerSecond) * float64(time.Second))
	} else {
		// Sin recarga configurada, no hay forma de recuperar tokens.
		wait = time.Hour
	}
	return false, wait
}

// sweepLocked elimina buckets cuya `last` es mas vieja que
// 2*CleanupInterval. El caller debe ya tener tomado l.mu.
func (l *limiter) sweepLocked(now time.Time) {
	cutoff := now.Add(-2 * l.cfg.CleanupInterval)
	for k, b := range l.buckets {
		if b.last.Before(cutoff) {
			delete(l.buckets, k)
		}
	}
}

// clientIP extrae la IP de cliente del request priorizando el primer
// valor de X-Forwarded-For, luego X-Real-IP, y finalmente RemoteAddr
// (despojado del puerto). Devuelve "unknown" si todo falla.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// El primer item de la lista es el cliente original.
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		if v := strings.TrimSpace(xff); v != "" {
			return v
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if r.RemoteAddr != "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
			return host
		}
		return r.RemoteAddr
	}
	return "unknown"
}
