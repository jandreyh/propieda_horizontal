package middleware_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/saas-ph/api/internal/platform/errors"
	"github.com/saas-ph/api/internal/platform/middleware"
)

// silentLogger devuelve un slog.Logger que descarta toda salida; los tests
// no necesitan inspeccionar el log, solo verificar la respuesta HTTP.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRecovery_GenericPanic_ReturnsProblemJSON500(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(silentLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/explode", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusInternalServerError)
	}

	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content-type: got %q, want prefix application/problem+json", got)
	}
}

func TestRecovery_GenericPanic_BodyContainsInternalAndStatus(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(silentLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/explode", nil)
	handler.ServeHTTP(rr, req)

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v (raw=%q)", err, rr.Body.String())
	}

	typeStr, _ := body["type"].(string)
	if !strings.Contains(typeStr, "internal") {
		t.Fatalf("type: got %q, want substring \"internal\"", typeStr)
	}

	// json.Unmarshal decodifica numeros como float64 por defecto.
	statusNum, ok := body["status"].(float64)
	if !ok {
		t.Fatalf("status field not numeric: %#v", body["status"])
	}
	if int(statusNum) != http.StatusInternalServerError {
		t.Fatalf("body.status: got %d, want %d", int(statusNum), http.StatusInternalServerError)
	}
}

func TestRecovery_PanicWithProblem_RespectsStatus(t *testing.T) {
	t.Parallel()

	want := apperrors.Problem{
		Type:   "urn:ph:problem:validation",
		Title:  "Unprocessable Entity",
		Status: http.StatusUnprocessableEntity,
		Detail: "campo invalido",
	}

	handler := middleware.Recovery(silentLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(want)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/things", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}

	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content-type: got %q, want prefix application/problem+json", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v (raw=%q)", err, rr.Body.String())
	}

	if got, _ := body["type"].(string); got != want.Type {
		t.Fatalf("body.type: got %q, want %q", got, want.Type)
	}
	if got, _ := body["title"].(string); got != want.Title {
		t.Fatalf("body.title: got %q, want %q", got, want.Title)
	}
	if got, _ := body["status"].(float64); int(got) != want.Status {
		t.Fatalf("body.status: got %d, want %d", int(got), want.Status)
	}
}

func TestRecovery_AbortHandler_IsRePanicked(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(silentLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/abort", nil)

	rec := func() (got any) {
		defer func() {
			got = recover()
		}()
		handler.ServeHTTP(rr, req)
		return nil
	}()

	if rec == nil {
		t.Fatal("expected panic to propagate, got none")
	}
	err, ok := rec.(error)
	if !ok || !errors.Is(err, http.ErrAbortHandler) {
		t.Fatalf("re-panic value: got %#v, want http.ErrAbortHandler", rec)
	}
}
