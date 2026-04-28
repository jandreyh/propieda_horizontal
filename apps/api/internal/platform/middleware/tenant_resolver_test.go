package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	dbpkg "github.com/saas-ph/api/internal/platform/db"
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
	// Devolvemos pool nil intencionalmente: el middleware solo lo
	// inyecta en el contexto, no lo desreferencia. Los tests verifican
	// el cableado, no la conexion real.
	return meta, nil, nil
}

// silentLogger devuelve un slog.Logger que descarta toda salida.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// tenantCapturingHandler permite capturar el tenant que ve el handler
// downstream y reportar si fue invocado.
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

func TestTenantResolver_ResolvesBySubdomain(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	reg.metas["acacias"] = dbpkg.TenantMetadata{
		ID: "tenant-1", Slug: "acacias", DisplayName: "Acacias",
	}

	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(cap.handler())

	req := httptest.NewRequest(http.MethodGet, "http://acacias.ph.localhost/things", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !cap.called {
		t.Fatal("downstream handler was not called")
	}
	if cap.err != nil {
		t.Fatalf("downstream FromCtx error: %v", cap.err)
	}
	if cap.tenant == nil || cap.tenant.Slug != "acacias" || cap.tenant.ID != "tenant-1" {
		t.Fatalf("tenant in ctx mismatch: got %+v", cap.tenant)
	}
	if reg.lastSlugSeen != "acacias" {
		t.Fatalf("registry last slug: want acacias, got %q", reg.lastSlugSeen)
	}
}

func TestTenantResolver_HeaderOverridesSubdomain(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	reg.metas["mobile-tenant"] = dbpkg.TenantMetadata{
		ID: "tenant-mobile", Slug: "mobile-tenant", DisplayName: "Mobile",
	}

	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(cap.handler())

	// El subdominio apunta a otro slug que NO existe en el registry; el
	// header debe ganar y por tanto el lookup tiene que ir contra
	// "mobile-tenant".
	req := httptest.NewRequest(http.MethodGet, "http://otro.ph.localhost/things", nil)
	req.Header.Set(HeaderTenantSlug, "mobile-tenant")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if reg.lastSlugSeen != "mobile-tenant" {
		t.Fatalf("expected header slug to win, registry saw %q", reg.lastSlugSeen)
	}
	if cap.tenant == nil || cap.tenant.Slug != "mobile-tenant" {
		t.Fatalf("ctx tenant mismatch: got %+v", cap.tenant)
	}
}

func TestTenantResolver_InvalidSlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		host   string
		header string
	}{
		{name: "uppercase_subdomain", host: "ACME.ph.localhost"}, // se normaliza pero acepta? probemos invalido
		{name: "header_with_underscore", host: "x.ph.localhost", header: "bad_slug"},
		{name: "host_equals_base", host: "ph.localhost"},
		{name: "host_outside_base", host: "tenant.example.com"},
		{name: "header_with_space", host: "x.ph.localhost", header: "bad slug"},
		{name: "header_starting_with_dash", host: "x.ph.localhost", header: "-bad"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := newFakeRegistry()
			cap := &tenantCapturingHandler{}
			srv := TenantResolver(TenantResolverConfig{
				Registry:   reg,
				BaseDomain: "ph.localhost",
				Logger:     silentLogger(),
			})(cap.handler())

			req := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/things", nil)
			req.Host = tc.host
			if tc.header != "" {
				req.Header.Set(HeaderTenantSlug, tc.header)
			}
			rr := httptest.NewRecorder()

			srv.ServeHTTP(rr, req)

			// Caso especial: ACME.ph.localhost se lowercased a
			// "acme.ph.localhost" que es valido. Si llegamos aqui con un
			// 200 lo aceptamos siempre que el handler no haya visto
			// tenant inexistente — pero en este test el registry esta
			// vacio, asi que esperamos 4xx.
			if rr.Code < 400 {
				t.Fatalf("expected error status, got %d body=%s",
					rr.Code, rr.Body.String())
			}
			if cap.called {
				t.Fatalf("downstream handler should NOT be called on slug error, status=%d", rr.Code)
			}
			// Verifica content-type problem+json.
			if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
				t.Fatalf("expected problem+json, got %q", ct)
			}
			// Para los casos puramente de formato esperamos 400. Para
			// "uppercase_subdomain" el slug normalizado es valido y el
			// registry vacio responde 404.
			var p struct {
				Status int `json:"status"`
			}
			_ = json.NewDecoder(rr.Body).Decode(&p)
			switch tc.name {
			case "uppercase_subdomain":
				if p.Status != http.StatusNotFound {
					t.Fatalf("expected 404 for normalized-but-unknown slug, got %d", p.Status)
				}
			default:
				if p.Status != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", p.Status)
				}
			}
		})
	}
}

func TestTenantResolver_TenantNotFound(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry() // vacio -> ErrTenantNotFound
	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(cap.handler())

	req := httptest.NewRequest(http.MethodGet, "http://desconocido.ph.localhost/things", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	if cap.called {
		t.Fatal("downstream handler should NOT be called on tenant-not-found")
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("expected problem+json, got %q", ct)
	}
}

func TestTenantResolver_RegistryInternalError(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	reg.errToReturn = errors.New("boom")
	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(cap.handler())

	req := httptest.NewRequest(http.MethodGet, "http://acacias.ph.localhost/things", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	if cap.called {
		t.Fatal("downstream handler should NOT be called on registry error")
	}
}

func TestTenantResolver_SkipBypassesResolution(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
		Skip: func(r *http.Request) bool {
			return strings.HasPrefix(r.URL.Path, "/health") ||
				strings.HasPrefix(r.URL.Path, "/superadmin/")
		},
	})(cap.handler())

	// Path /health: aunque el host no tiene subdominio valido, el
	// middleware debe saltarse la resolucion.
	req := httptest.NewRequest(http.MethodGet, "http://ph.localhost/health", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !cap.called {
		t.Fatal("downstream handler should be called when Skip returns true")
	}
	if cap.err == nil {
		t.Fatalf("expected ErrNoTenant when Skip bypasses, got tenant %+v", cap.tenant)
	}
	if !errors.Is(cap.err, tenantctx.ErrNoTenant) {
		t.Fatalf("expected ErrNoTenant, got %v", cap.err)
	}
	if reg.calls != 0 {
		t.Fatalf("registry should not be called when Skip returns true, got %d calls", reg.calls)
	}
}

func TestTenantResolver_DownstreamReadsTenantFromCtx(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	reg.metas["villavo"] = dbpkg.TenantMetadata{
		ID: "tenant-uuid-9", Slug: "villavo", DisplayName: "Villavicencio",
	}

	var (
		seenID, seenSlug, seenName string
		seenErr                    error
	)
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t, err := tenantctx.FromCtx(r.Context())
		if err != nil {
			seenErr = err
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		seenID, seenSlug, seenName = t.ID, t.Slug, t.DisplayName
		w.WriteHeader(http.StatusOK)
	})

	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(downstream)

	req := httptest.NewRequest(http.MethodGet, "http://villavo.ph.localhost/api/things", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if seenErr != nil {
		t.Fatalf("downstream FromCtx error: %v", seenErr)
	}
	if seenID != "tenant-uuid-9" || seenSlug != "villavo" || seenName != "Villavicencio" {
		t.Fatalf("tenant fields mismatch: id=%q slug=%q name=%q",
			seenID, seenSlug, seenName)
	}
}

func TestTenantResolver_HostWithPortIsTrimmed(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	reg.metas["yopal"] = dbpkg.TenantMetadata{ID: "id-y", Slug: "yopal"}

	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   reg,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(cap.handler())

	req := httptest.NewRequest(http.MethodGet, "http://yopal.ph.localhost:8080/x", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if cap.tenant == nil || cap.tenant.Slug != "yopal" {
		t.Fatalf("expected slug yopal, got %+v", cap.tenant)
	}
}

func TestTenantResolver_DefaultHeaderName(t *testing.T) {
	t.Parallel()

	reg := newFakeRegistry()
	reg.metas["m1"] = dbpkg.TenantMetadata{ID: "id-m1", Slug: "m1"}

	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry: reg,
		// BaseDomain vacio: la unica via es el header.
		Logger: silentLogger(),
	})(cap.handler())

	req := httptest.NewRequest(http.MethodGet, "http://anything/here", nil)
	req.Header.Set(HeaderTenantSlug, "m1")
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if cap.tenant == nil || cap.tenant.Slug != "m1" {
		t.Fatalf("ctx tenant mismatch: got %+v", cap.tenant)
	}
}

func TestTenantResolver_NilRegistryReturns500(t *testing.T) {
	t.Parallel()

	cap := &tenantCapturingHandler{}
	srv := TenantResolver(TenantResolverConfig{
		Registry:   nil,
		BaseDomain: "ph.localhost",
		Logger:     silentLogger(),
	})(cap.handler())

	req := httptest.NewRequest(http.MethodGet, "http://acacias.ph.localhost/x", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	if cap.called {
		t.Fatal("downstream should not be called when registry is nil")
	}
}
