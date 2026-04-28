package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/tenant_config/domain"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
	tcfghttp "github.com/saas-ph/api/internal/modules/tenant_config/interfaces/http"
)

// --- in-memory repos ---

type memSettings struct {
	items map[string]entities.Setting
}

func newMemSettings() *memSettings {
	return &memSettings{items: map[string]entities.Setting{
		"contact.email": {
			ID: "00000000-0000-0000-0000-000000000001", Key: "contact.email",
			Value: []byte(`"admin@conjunto.test"`), Status: entities.SettingStatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now(), Version: 1,
		},
	}}
}

func (m *memSettings) List(ctx context.Context, f domain.ListSettingsFilter) ([]entities.Setting, int64, error) {
	out := make([]entities.Setting, 0, len(m.items))
	for _, v := range m.items {
		if v.IsArchived() {
			continue
		}
		if f.Category != "" && v.Category != f.Category {
			continue
		}
		out = append(out, v)
	}
	return out, int64(len(out)), nil
}

func (m *memSettings) Get(ctx context.Context, key string) (entities.Setting, error) {
	v, ok := m.items[key]
	if !ok || v.IsArchived() {
		return entities.Setting{}, domain.ErrSettingNotFound
	}
	return v, nil
}

func (m *memSettings) Upsert(ctx context.Context, in domain.UpsertSettingInput) (entities.Setting, error) {
	cur, ok := m.items[in.Key]
	if !ok {
		cur = entities.Setting{ID: in.Key + "-id", Key: in.Key, CreatedAt: time.Now(), Version: 0}
	}
	cur.Value = in.Value
	if in.Description != "" {
		cur.Description = in.Description
	}
	if in.Category != "" {
		cur.Category = in.Category
	}
	cur.Status = entities.SettingStatusActive
	cur.UpdatedAt = time.Now()
	cur.Version++
	m.items[in.Key] = cur
	return cur, nil
}

func (m *memSettings) Archive(ctx context.Context, key, actorID string) (entities.Setting, error) {
	v, ok := m.items[key]
	if !ok || v.IsArchived() {
		return entities.Setting{}, domain.ErrSettingNotFound
	}
	now := time.Now()
	v.Status = entities.SettingStatusArchived
	v.DeletedAt = &now
	v.UpdatedAt = now
	v.Version++
	m.items[key] = v
	return v, nil
}

type memBranding struct {
	cur entities.Branding
}

func newMemBranding() *memBranding {
	return &memBranding{cur: entities.Branding{
		ID: "11111111-1111-1111-1111-111111111111", DisplayName: "Default",
		Timezone: "America/Bogota", Locale: "es-CO", Status: entities.BrandingStatusActive,
		CreatedAt: time.Now(), UpdatedAt: time.Now(), Version: 1,
	}}
}

func (m *memBranding) Get(ctx context.Context) (entities.Branding, error) {
	return m.cur, nil
}

func (m *memBranding) Update(ctx context.Context, in domain.UpdateBrandingInput) (entities.Branding, error) {
	if in.ExpectedVersion != m.cur.Version {
		return entities.Branding{}, domain.ErrVersionMismatch
	}
	m.cur.DisplayName = in.DisplayName
	m.cur.LogoURL = in.LogoURL
	m.cur.PrimaryColor = in.PrimaryColor
	m.cur.SecondaryColor = in.SecondaryColor
	m.cur.Timezone = in.Timezone
	m.cur.Locale = in.Locale
	m.cur.UpdatedAt = time.Now()
	m.cur.Version++
	return m.cur, nil
}

// --- helpers ---

func mountTest(t *testing.T) (*chi.Mux, *memSettings, *memBranding) {
	t.Helper()
	r := chi.NewRouter()
	s := newMemSettings()
	b := newMemBranding()
	tcfghttp.Mount(r, tcfghttp.Dependencies{
		SettingsRepo: s,
		BrandingRepo: b,
	})
	return r, s, b
}

// --- tests ---

func TestListSettings(t *testing.T) {
	r, _, _ := mountTest(t)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 || len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got total=%d items=%d", body.Total, len(body.Items))
	}
}

func TestGetSetting_OK(t *testing.T) {
	r, _, _ := mountTest(t)
	req := httptest.NewRequest(http.MethodGet, "/settings/contact.email", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	r, _, _ := mountTest(t)
	req := httptest.NewRequest(http.MethodGet, "/settings/missing.key", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:len("application/problem+json")] != "application/problem+json" {
		t.Fatalf("expected problem+json, got %q", ct)
	}
}

func TestPutSetting_OK(t *testing.T) {
	r, mem, _ := mountTest(t)
	body := []byte(`{"value": 42, "category": "general"}`)
	req := httptest.NewRequest(http.MethodPut, "/settings/visits.max", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, ok := mem.items["visits.max"]; !ok {
		t.Fatalf("setting not stored")
	}
}

func TestPutSetting_BadKey(t *testing.T) {
	r, _, _ := mountTest(t)
	body := []byte(`{"value": true}`)
	req := httptest.NewRequest(http.MethodPut, "/settings/BAD-KEY", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutSetting_InvalidJSONValue(t *testing.T) {
	r, _, _ := mountTest(t)
	body := []byte(`{"value": "not-quoted-properly"`)
	req := httptest.NewRequest(http.MethodPut, "/settings/x.y", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteSetting_OK(t *testing.T) {
	r, _, _ := mountTest(t)
	req := httptest.NewRequest(http.MethodDelete, "/settings/contact.email", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetBranding(t *testing.T) {
	r, _, _ := mountTest(t)
	req := httptest.NewRequest(http.MethodGet, "/branding", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["display_name"] != "Default" {
		t.Fatalf("got %v", resp["display_name"])
	}
}

func TestPutBranding_OK(t *testing.T) {
	r, _, mem := mountTest(t)
	body := []byte(`{
		"display_name": "Acacias",
		"timezone": "America/Bogota",
		"locale": "es-CO",
		"primary_color": "#112233",
		"expected_version": 1
	}`)
	req := httptest.NewRequest(http.MethodPut, "/branding", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if mem.cur.Version != 2 {
		t.Fatalf("version not bumped: %d", mem.cur.Version)
	}
}

func TestPutBranding_VersionMismatch(t *testing.T) {
	r, _, _ := mountTest(t)
	body := []byte(`{
		"display_name": "Acacias",
		"timezone": "UTC",
		"locale": "es-CO",
		"expected_version": 99
	}`)
	req := httptest.NewRequest(http.MethodPut, "/branding", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutBranding_BadColor(t *testing.T) {
	r, _, _ := mountTest(t)
	body := []byte(`{
		"display_name": "X",
		"timezone": "UTC",
		"locale": "es-CO",
		"primary_color": "blue",
		"expected_version": 1
	}`)
	req := httptest.NewRequest(http.MethodPut, "/branding", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGuard_BlocksWhenSet(t *testing.T) {
	r := chi.NewRouter()
	tcfghttp.Mount(r, tcfghttp.Dependencies{
		SettingsRepo: newMemSettings(),
		BrandingRepo: newMemBranding(),
	}, tcfghttp.WithGuard(func(ns string) func(http.Handler) http.Handler {
		// Guard que siempre niega para confirmar que se aplica.
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			})
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected guard to deny, got %d", rec.Code)
	}
}
