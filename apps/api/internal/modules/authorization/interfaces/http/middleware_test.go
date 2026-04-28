package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeResolver implementa permissionResolver con un set predefinido por
// userID. Soporta opcionalmente un error.
type fakeResolver struct {
	perms map[string][]string
	err   error
	calls int
}

func (f *fakeResolver) Permissions(_ context.Context, userID string) ([]string, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.perms[userID], nil
}

func newOK() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}
}

func TestRequirePermission_Allows(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{perms: map[string][]string{"u-1": {"package.read"}}}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequirePermission("package.read")(newOK())

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequirePermission_Forbidden(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{perms: map[string][]string{"u-1": {"visit.read"}}}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequirePermission("package.deliver")(newOK())

	req := httptest.NewRequest(http.MethodPost, "/towers/1/packages", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing permission: package.deliver") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestRequirePermission_Wildcard(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{perms: map[string][]string{"u-1": {"package.*"}}}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequirePermission("package.deliver")(newOK())
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with wildcard grant; got %d", rec.Code)
	}
}

func TestRequirePermission_Unauthenticated(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequirePermission("package.read")(newOK())
	req := httptest.NewRequest(http.MethodGet, "/x", nil) // no user_id in ctx
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401; got %d", rec.Code)
	}
}

func TestRequirePermission_ResolverError(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{err: errors.New("db down")}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequirePermission("package.read")(newOK())
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d; want 500", rec.Code)
	}
}

func TestRequirePermission_CachesBySession(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{perms: map[string][]string{"u-1": {"package.read"}}}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequirePermission("package.read")(newOK())
	doRequest := func() {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		ctx := WithUserID(req.Context(), "u-1")
		ctx = WithSessionID(ctx, "session-X")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want 200", rec.Code)
		}
	}
	doRequest()
	doRequest()
	doRequest()

	if resolver.calls != 1 {
		t.Fatalf("resolver called %d times; want 1 (cache hit)", resolver.calls)
	}
}

func TestPermissionCache_TTLExpiry(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	cache := NewPermissionCache(60*time.Second, clock)

	cache.Set("k1", []string{"a.b"})
	if _, ok := cache.Get("k1"); !ok {
		t.Fatal("expected hit before expiry")
	}
	now = now.Add(61 * time.Second)
	if _, ok := cache.Get("k1"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestPermissionCache_Invalidate(t *testing.T) {
	t.Parallel()
	cache := NewPermissionCache(60*time.Second, time.Now)
	cache.Set("k1", []string{"a.b"})
	cache.Invalidate("k1")
	if _, ok := cache.Get("k1"); ok {
		t.Fatal("expected miss after Invalidate")
	}
}

func TestPermissionCache_DisabledTTL(t *testing.T) {
	t.Parallel()
	cache := NewPermissionCache(0, time.Now)
	cache.Set("k1", []string{"a.b"})
	if _, ok := cache.Get("k1"); ok {
		t.Fatal("expected miss when TTL is disabled")
	}
}

func TestRequireAnyPermission(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{perms: map[string][]string{"u-1": {"visit.read"}}}
	mw := MiddlewareConfig{Resolver: resolver, Cache: NewPermissionCache(60*time.Second, time.Now)}

	handler := mw.RequireAnyPermission("package.read", "visit.read")(newOK())
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(WithUserID(req.Context(), "u-1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
}
