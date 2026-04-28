package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock construye un par (now, advance) sobre un instante mutable
// protegido por mutex, util para inyectar en RateLimitConfig.Now.
//
// El closure `now` devuelve el instante actual del reloj falso, y
// `advance` lo adelanta en la duracion indicada. Ambas operaciones son
// goroutine-safe.
func fakeClock(start time.Time) (now func() time.Time, advance func(d time.Duration)) {
	var mu sync.Mutex
	current := start
	now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return current
	}
	advance = func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		current = current.Add(d)
	}
	return now, advance
}

// okHandler es un handler trivial que responde 200/OK, util como
// downstream del middleware en tests.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
})

// doRequest envia una request sintetica al handler con el RemoteAddr
// solicitado y devuelve el ResponseRecorder.
func doRequest(t *testing.T, h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestRateLimit_AllowsBurstAndRejectsExcess(t *testing.T) {
	t.Parallel()

	now, _ := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             2,
		Now:               now,
	})
	h := mw(okHandler)

	// Primeras dos requests deben pasar (consume Burst inicial).
	for i := 0; i < 2; i++ {
		rr := doRequest(t, h, "10.0.0.1:1234")
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	// Tercera request en el mismo instante debe ser rechazada.
	rr := doRequest(t, h, "10.0.0.1:1234")
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("third request: expected 429, got %d", rr.Code)
	}
}

func TestRateLimit_RecoversAfterOneSecond(t *testing.T) {
	t.Parallel()

	now, advance := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             2,
		Now:               now,
	})
	h := mw(okHandler)

	// Agota el burst.
	for i := 0; i < 2; i++ {
		if rr := doRequest(t, h, "10.0.0.2:1234"); rr.Code != http.StatusOK {
			t.Fatalf("burst request %d: expected 200, got %d", i+1, rr.Code)
		}
	}
	// Tercera rechazada.
	if rr := doRequest(t, h, "10.0.0.2:1234"); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before refill, got %d", rr.Code)
	}

	// Avanza 1s -> recupera 1 token.
	advance(1 * time.Second)
	if rr := doRequest(t, h, "10.0.0.2:1234"); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after refill, got %d", rr.Code)
	}
	// Y la siguiente vuelve a fallar (token recien recuperado ya consumido).
	if rr := doRequest(t, h, "10.0.0.2:1234"); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after consuming refill, got %d", rr.Code)
	}
}

func TestRateLimit_429IncludesRetryAfter(t *testing.T) {
	t.Parallel()

	now, _ := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		Now:               now,
	})
	h := mw(okHandler)

	// Consume el unico token disponible.
	if rr := doRequest(t, h, "10.0.0.3:1234"); rr.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr.Code)
	}

	// Segunda request debe ser rechazada con Retry-After.
	rr := doRequest(t, h, "10.0.0.3:1234")
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
	ra := rr.Header().Get("Retry-After")
	if ra == "" {
		t.Fatal("missing Retry-After header on 429 response")
	}
	secs, err := strconv.Atoi(ra)
	if err != nil {
		t.Fatalf("Retry-After is not an integer: %q (%v)", ra, err)
	}
	if secs < 1 {
		t.Fatalf("expected Retry-After >= 1, got %d", secs)
	}
}

func TestRateLimit_429BodyIsProblemJSON(t *testing.T) {
	t.Parallel()

	now, _ := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		Now:               now,
	})
	h := mw(okHandler)

	// Consume el token.
	if rr := doRequest(t, h, "10.0.0.4:1234"); rr.Code != http.StatusOK {
		t.Fatalf("setup: expected 200, got %d", rr.Code)
	}

	rr := doRequest(t, h, "10.0.0.4:1234")
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct == "" || !containsCI(ct, "application/problem+json") {
		t.Fatalf("expected application/problem+json content-type, got %q", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not valid JSON: %v (raw=%q)", err, rr.Body.String())
	}
	statusVal, ok := body["status"]
	if !ok {
		t.Fatalf("body missing `status` field: %v", body)
	}
	// json.Unmarshal decodifica numeros como float64.
	if f, isFloat := statusVal.(float64); !isFloat || int(f) != http.StatusTooManyRequests {
		t.Fatalf("expected status=429 in body, got %v (%T)", statusVal, statusVal)
	}
	if detail, _ := body["detail"].(string); detail != "rate limit exceeded" {
		t.Fatalf("expected detail=\"rate limit exceeded\", got %q", detail)
	}
}

func TestRateLimit_ConcurrentBurstCap(t *testing.T) {
	t.Parallel()

	const burst = 10
	const goroutines = 200

	now, _ := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		// Tasa muy baja para que la recarga sea despreciable durante el
		// test: solo el Burst inicial debe pasar.
		RequestsPerSecond: 0.0001,
		Burst:             burst,
		Now:               now,
	})
	h := mw(okHandler)

	var ok int64
	var rejected int64

	var start sync.WaitGroup
	start.Add(1)
	var done sync.WaitGroup
	done.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer done.Done()
			start.Wait()
			rr := doRequest(t, h, "10.0.0.5:1234")
			switch rr.Code {
			case http.StatusOK:
				atomic.AddInt64(&ok, 1)
			case http.StatusTooManyRequests:
				atomic.AddInt64(&rejected, 1)
			}
		}()
	}

	start.Done()
	done.Wait()

	finalOK := atomic.LoadInt64(&ok)
	finalRej := atomic.LoadInt64(&rejected)

	if finalOK > int64(burst) {
		t.Fatalf("burst cap violated: ok=%d > burst=%d", finalOK, burst)
	}
	if finalOK+finalRej != int64(goroutines) {
		t.Fatalf("missing responses: ok=%d rejected=%d total=%d (want %d)",
			finalOK, finalRej, finalOK+finalRej, goroutines)
	}
	// Sanity: al menos hubo algunas exitosas (sino algo mas falla).
	if finalOK == 0 {
		t.Fatal("expected at least one successful request")
	}
}

func TestRateLimit_KeysAreIsolated(t *testing.T) {
	t.Parallel()

	now, _ := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		Now:               now,
	})
	h := mw(okHandler)

	// Cada IP tiene su propio bucket: ambas pueden consumir su token.
	if rr := doRequest(t, h, "10.0.0.6:1234"); rr.Code != http.StatusOK {
		t.Fatalf("ip6 first: expected 200, got %d", rr.Code)
	}
	if rr := doRequest(t, h, "10.0.0.7:1234"); rr.Code != http.StatusOK {
		t.Fatalf("ip7 first: expected 200, got %d", rr.Code)
	}
	// Y cada una, por separado, hace 429 en su segunda llamada.
	if rr := doRequest(t, h, "10.0.0.6:1234"); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("ip6 second: expected 429, got %d", rr.Code)
	}
	if rr := doRequest(t, h, "10.0.0.7:1234"); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("ip7 second: expected 429, got %d", rr.Code)
	}
}

func TestRateLimit_XForwardedForUsedAsKey(t *testing.T) {
	t.Parallel()

	now, _ := fakeClock(time.Unix(1_700_000_000, 0))
	mw := RateLimit(RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		Now:               now,
	})
	h := mw(okHandler)

	send := func(xff, remote string) int {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = remote
		req.Header.Set("X-Forwarded-For", xff)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr.Code
	}

	// Misma XFF, distintos RemoteAddr -> deben compartir bucket.
	if got := send("203.0.113.7", "10.0.0.10:1111"); got != http.StatusOK {
		t.Fatalf("first XFF request: expected 200, got %d", got)
	}
	if got := send("203.0.113.7", "10.0.0.11:2222"); got != http.StatusTooManyRequests {
		t.Fatalf("second XFF request: expected 429, got %d", got)
	}
}

// containsCI hace contains case-insensitive sin pull de strings.ToLower
// global (solo sobre los argumentos pasados).
func containsCI(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			a := haystack[i+j]
			b := needle[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
