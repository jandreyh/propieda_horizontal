package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// captureHandler observa el id presente en el contexto y lo escribe al
// body para asertarlo desde los tests.
func captureHandler(seen *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFrom(r.Context())
		if seen != nil {
			*seen = id
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(id))
	})
}

func TestRequestID_PropagatesValidClientID(t *testing.T) {
	t.Parallel()

	const clientID = "abc12345-valid-id_v1"

	var got string
	srv := RequestID(captureHandler(&got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, clientID)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if got != clientID {
		t.Fatalf("ctx request id mismatch: want %q, got %q", clientID, got)
	}
	if h := rr.Header().Get(HeaderRequestID); h != clientID {
		t.Fatalf("response header mismatch: want %q, got %q", clientID, h)
	}
	if body := rr.Body.String(); body != clientID {
		t.Fatalf("body mismatch: want %q, got %q", clientID, body)
	}
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	t.Parallel()

	var got string
	srv := RequestID(captureHandler(&got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if got == "" {
		t.Fatal("expected generated id in context, got empty")
	}
	header := rr.Header().Get(HeaderRequestID)
	if header == "" {
		t.Fatal("expected generated id in response header, got empty")
	}
	if header != got {
		t.Fatalf("header/context mismatch: header=%q ctx=%q", header, got)
	}
	if !isValidRequestID(got) {
		t.Fatalf("generated id %q does not match expected format", got)
	}
}

func TestRequestID_ReplacesInvalidValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "too_short", in: "abc"},
		{name: "invalid_chars", in: "id with spaces!"},
		{name: "too_long", in: strings.Repeat("a", 129)},
		{name: "bad_symbol", in: "abcdefgh$$$"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got string
			srv := RequestID(captureHandler(&got))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.in != "" {
				req.Header.Set(HeaderRequestID, tc.in)
			}
			rr := httptest.NewRecorder()

			srv.ServeHTTP(rr, req)

			if got == "" {
				t.Fatal("expected replacement id, got empty")
			}
			if got == tc.in {
				t.Fatalf("expected replacement id, but got original %q", tc.in)
			}
			if !isValidRequestID(got) {
				t.Fatalf("replacement id %q does not match expected format", got)
			}
			if rr.Header().Get(HeaderRequestID) != got {
				t.Fatalf("response header not aligned with ctx id: header=%q ctx=%q",
					rr.Header().Get(HeaderRequestID), got)
			}
		})
	}
}

func TestRequestID_ConcurrentRequestsAreIndependent(t *testing.T) {
	t.Parallel()

	const n = 64

	srv := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(RequestIDFrom(r.Context())))
	}))

	var (
		mu  sync.Mutex
		ids = make(map[string]struct{}, n)
		wg  sync.WaitGroup
	)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, req)

			id := rr.Body.String()
			if id == "" {
				t.Errorf("empty id in concurrent request")
				return
			}
			if rr.Header().Get(HeaderRequestID) != id {
				t.Errorf("header/body mismatch: header=%q body=%q",
					rr.Header().Get(HeaderRequestID), id)
				return
			}

			mu.Lock()
			ids[id] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(ids) != n {
		t.Fatalf("expected %d unique ids, got %d", n, len(ids))
	}
}

func TestRequestIDFrom_EmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if id := RequestIDFrom(req.Context()); id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}
}
