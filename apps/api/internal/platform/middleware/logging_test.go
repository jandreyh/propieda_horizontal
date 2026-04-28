package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newBufferLogger construye un *slog.Logger que escribe JSON en un buffer
// para que los tests puedan parsear la salida sin tocar stdout.
func newBufferLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// decodeLogLines parsea la salida JSON-line del logger en mapas.
func decodeLogLines(tb testing.TB, buf *bytes.Buffer) []map[string]any {
	tb.Helper()
	out := make([]map[string]any, 0)
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			tb.Fatalf("salida no es JSON valido: %v\nlinea=%q", err, line)
		}
		out = append(out, m)
	}
	return out
}

func TestLogging_EmitsLogWithRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	const requestID = "req-test-123"
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/things?x=1", http.NoBody)
	ctx := WithRequestID(context.Background(), requestID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code: want 200 got %d", rec.Code)
	}

	lines := decodeLogLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("se esperaba 1 linea de log, hay %d: %v", len(lines), lines)
	}
	entry := lines[0]

	if got := entry["method"]; got != http.MethodGet {
		t.Errorf("method: want GET got %v", got)
	}
	if got := entry["status"]; got != float64(200) {
		t.Errorf("status: want 200 got %v", got)
	}
	if got := entry["request_id"]; got != requestID {
		t.Errorf("request_id: want %q got %v", requestID, got)
	}
	if got := entry["path"]; got != "/api/things" {
		t.Errorf("path: want /api/things got %v", got)
	}
	if got := entry["query"]; got != "x=1" {
		t.Errorf("query: want x=1 got %v", got)
	}
	if got := entry["msg"]; got != "http_request" {
		t.Errorf("msg: want http_request got %v", got)
	}
	if _, ok := entry["duration_ms"].(float64); !ok {
		t.Errorf("duration_ms: se esperaba float64, got %T", entry["duration_ms"])
	}
	if got := entry["bytes"]; got != float64(2) {
		t.Errorf("bytes: want 2 got %v", got)
	}
}

func TestLogging_SkipsHealthByDefault(t *testing.T) {
	// No usamos t.Parallel() porque t.Setenv obliga a serializar.

	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	// Nos aseguramos de que el env var NO este activado para este test.
	t.Setenv(envLogHealthchecks, "")

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/ready"} {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status code want 200 got %d", path, rec.Code)
		}
	}

	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Fatalf("se esperaba que /health y /ready no emitieran log, got: %q", got)
	}
}

func TestLogging_LogsHealthWhenEnvEnabled(t *testing.T) {
	// No usamos t.Parallel() porque t.Setenv obliga a serializar.
	t.Setenv(envLogHealthchecks, "true")

	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	lines := decodeLogLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("se esperaba 1 linea de log con LOG_HEALTHCHECKS=true, hay %d", len(lines))
	}
	if got := lines[0]["path"]; got != "/health" {
		t.Errorf("path: want /health got %v", got)
	}
}

func TestResponseRecorder_CapturesStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/things/42", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status code expuesto al cliente: want 204 got %d", rec.Code)
	}

	lines := decodeLogLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("se esperaba 1 linea de log, hay %d", len(lines))
	}
	entry := lines[0]
	if got := entry["status"]; got != float64(http.StatusNoContent) {
		t.Errorf("recorder status: want 204 got %v", got)
	}
	if got := entry["level"]; got != "INFO" {
		t.Errorf("level: want INFO got %v", got)
	}
}

func TestLogging_LevelByStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		level  string
	}{
		{"server error", http.StatusInternalServerError, "ERROR"},
		{"client error", http.StatusBadRequest, "WARN"},
		{"ok", http.StatusOK, "INFO"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := newBufferLogger(&buf)

			handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}))

			req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
			handler.ServeHTTP(httptest.NewRecorder(), req)

			lines := decodeLogLines(t, &buf)
			if len(lines) != 1 {
				t.Fatalf("se esperaba 1 linea, hay %d", len(lines))
			}
			if got := lines[0]["level"]; got != tc.level {
				t.Errorf("level: want %s got %v", tc.level, got)
			}
		})
	}
}

func TestNewLogger_FormatAndLevel(t *testing.T) {
	t.Parallel()

	if l := NewLogger("json", "debug"); l == nil {
		t.Fatal("NewLogger json/debug: nil")
	}
	if l := NewLogger("text", "warn"); l == nil {
		t.Fatal("NewLogger text/warn: nil")
	}
	if l := NewLogger("", ""); l == nil {
		t.Fatal("NewLogger defaults: nil")
	}
}

func TestLogging_PrefersXForwardedFor(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.10")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	lines := decodeLogLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("se esperaba 1 linea de log, hay %d", len(lines))
	}
	if got, want := lines[0]["client_ip"], "203.0.113.5"; got != want {
		t.Errorf("client_ip: want %q got %v", want, got)
	}
}
