package middleware

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	dbpkg "github.com/saas-ph/api/internal/platform/db"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// fakeRegistry es una implementacion en memoria de tenantLookup para
// poder testear el middleware sin abrir Postgres.
type fakeRegistry struct {
	metas        map[string]dbpkg.TenantMetadata
	errToReturn  error
	calls        int
	lastSlugSeen string
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{metas: make(map[string]dbpkg.TenantMetadata)}
}

func (f *fakeRegistry) Get(_ context.Context, slug string) (dbpkg.TenantMetadata, *pgxpool.Pool, error) {
	f.calls++
	f.lastSlugSeen = slug
	if f.errToReturn != nil {
		return dbpkg.TenantMetadata{}, nil, f.errToReturn
	}
	meta, ok := f.metas[slug]
	if !ok {
		return dbpkg.TenantMetadata{}, nil, dbpkg.ErrTenantNotFound
	}
	return meta, nil, nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

type tenantCapturingHandler struct {
	called bool
	tenant *tenantctx.Tenant
	err    error
}

func (h *tenantCapturingHandler) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.called = true
		h.tenant, h.err = tenantctx.FromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

// newTestSigner construye un Signer con clave efimera para tests.
func newTestSigner(t *testing.T) *jwtsign.Signer {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	signer, err := jwtsign.NewSigner(jwtsign.SignerConfig{
		KeyID: "test", PrivateKey: priv, PublicKey: pub,
	})
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer
}

// signClaimsToken emite un JWT con el sid+memberships+current_tenant indicados.
func signClaimsToken(t *testing.T, s *jwtsign.Signer, subject, currentTenant, sid string, memberships []jwtsign.MembershipClaim) string {
	t.Helper()
	tok, err := s.SignPlatform(subject, sid, currentTenant, memberships, []string{"pwd"}, time.Minute)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

// chainAuthThenResolver corre PlatformAuth seguido de TenantResolver.
func chainAuthThenResolver(signer *jwtsign.Signer, reg tenantLookup, h http.Handler) http.Handler {
	return PlatformAuth(PlatformAuthConfig{Signer: signer})(
		TenantResolver(TenantResolverConfig{Registry: reg, Logger: silentLogger()})(h),
	)
}

func TestTenantResolver_Success_FromJWT(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	reg := newFakeRegistry()
	reg.metas["acacias"] = dbpkg.TenantMetadata{ID: "tenant-1", Slug: "acacias", DisplayName: "Acacias"}

	cap := &tenantCapturingHandler{}
	srv := chainAuthThenResolver(signer, reg, cap.handler())

	tok := signClaimsToken(t, signer, "user-1", "acacias", "real-sid",
		[]jwtsign.MembershipClaim{{TenantID: "tenant-1", TenantSlug: "acacias", TenantName: "Acacias"}})

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !cap.called || cap.tenant == nil || cap.tenant.Slug != "acacias" {
		t.Errorf("tenant not injected: %+v", cap)
	}
}

func TestTenantResolver_NoToken_401(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	reg := newFakeRegistry()
	srv := chainAuthThenResolver(signer, reg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestTenantResolver_NoCurrentTenant_412(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	reg := newFakeRegistry()
	srv := chainAuthThenResolver(signer, reg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tok := signClaimsToken(t, signer, "user-1", "", "real-sid",
		[]jwtsign.MembershipClaim{{TenantSlug: "acacias"}})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d body=%s", rr.Code, rr.Body.String())
	}
	var p map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&p)
	if p["title"] != "Tenant Not Selected" {
		t.Errorf("unexpected title: %v", p["title"])
	}
}

func TestTenantResolver_NoMembership_403(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	reg := newFakeRegistry()
	reg.metas["acacias"] = dbpkg.TenantMetadata{ID: "tenant-1", Slug: "acacias", DisplayName: "Acacias"}
	srv := chainAuthThenResolver(signer, reg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// claims dicen current_tenant=acacias, pero memberships solo trae demo2.
	tok := signClaimsToken(t, signer, "user-1", "acacias", "real-sid",
		[]jwtsign.MembershipClaim{{TenantSlug: "demo2"}})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTenantResolver_TenantNotFound_404(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	reg := newFakeRegistry() // no metas → ErrTenantNotFound
	srv := chainAuthThenResolver(signer, reg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tok := signClaimsToken(t, signer, "user-1", "ghost", "real-sid",
		[]jwtsign.MembershipClaim{{TenantSlug: "ghost"}})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestTenantResolver_RegistryError_500(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	reg := newFakeRegistry()
	reg.errToReturn = errors.New("boom")
	srv := chainAuthThenResolver(signer, reg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tok := signClaimsToken(t, signer, "user-1", "acacias", "real-sid",
		[]jwtsign.MembershipClaim{{TenantSlug: "acacias"}})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestTenantResolver_NilRegistry_500(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)

	wrap := PlatformAuth(PlatformAuthConfig{Signer: signer})(
		TenantResolver(TenantResolverConfig{Registry: nil, Logger: silentLogger()})(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		),
	)

	tok := signClaimsToken(t, signer, "user-1", "acacias", "real-sid",
		[]jwtsign.MembershipClaim{{TenantSlug: "acacias"}})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	wrap.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestTenantResolver_Skip(t *testing.T) {
	t.Parallel()
	reg := newFakeRegistry()

	called := false
	mw := TenantResolver(TenantResolverConfig{
		Registry: reg,
		Logger:   silentLogger(),
		Skip:     func(*http.Request) bool { return true },
	})
	srv := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if !called {
		t.Error("Skip true should let request through without resolving")
	}
}

func TestPlatformAuth_PreAuthRejected(t *testing.T) {
	t.Parallel()
	signer := newTestSigner(t)
	mw := PlatformAuth(PlatformAuthConfig{Signer: signer})
	srv := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tok, err := signer.Sign("user-1", "", PreAuthSessionMarker, []string{"pre-auth:mfa"}, []string{"pwd"}, time.Minute)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for pre-auth token, got %d", rr.Code)
	}
}
