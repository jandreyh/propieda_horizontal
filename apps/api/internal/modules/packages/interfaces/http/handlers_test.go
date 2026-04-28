package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	pkghttp "github.com/saas-ph/api/internal/modules/packages/interfaces/http"
)

// --- in-memory fakes (minimo necesario para el test) ---

type stubPackages struct{}

func (s *stubPackages) Create(_ context.Context, _ domain.CreatePackageInput) (entities.Package, error) {
	return entities.Package{}, nil
}
func (s *stubPackages) GetByID(_ context.Context, _ string) (entities.Package, error) {
	return entities.Package{}, domain.ErrPackageNotFound
}
func (s *stubPackages) ListByUnit(_ context.Context, _ string) ([]entities.Package, error) {
	return nil, nil
}
func (s *stubPackages) ListByStatus(_ context.Context, _ entities.PackageStatus) ([]entities.Package, error) {
	return nil, nil
}
func (s *stubPackages) UpdateStatusOptimistic(_ context.Context, _ string, _ int32, _ entities.PackageStatus, _ string) (entities.Package, error) {
	return entities.Package{}, nil
}
func (s *stubPackages) Return(_ context.Context, _ string, _ int32, _ string) (entities.Package, error) {
	return entities.Package{}, nil
}
func (s *stubPackages) ListPendingReminder(_ context.Context) ([]entities.Package, error) {
	return nil, nil
}

type stubCategories struct{}

func (s *stubCategories) List(_ context.Context) ([]entities.PackageCategory, error) { return nil, nil }
func (s *stubCategories) GetByName(_ context.Context, _ string) (entities.PackageCategory, error) {
	return entities.PackageCategory{}, domain.ErrCategoryNotFound
}
func (s *stubCategories) GetByID(_ context.Context, _ string) (entities.PackageCategory, error) {
	return entities.PackageCategory{}, domain.ErrCategoryNotFound
}

type stubDeliveries struct{}

func (s *stubDeliveries) Record(_ context.Context, _ domain.RecordDeliveryInput) (entities.DeliveryEvent, error) {
	return entities.DeliveryEvent{}, nil
}

type stubOutbox struct{}

func (s *stubOutbox) Enqueue(_ context.Context, _ domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	return entities.OutboxEvent{}, nil
}
func (s *stubOutbox) LockPending(_ context.Context, _ int32) ([]entities.OutboxEvent, error) {
	return nil, nil
}
func (s *stubOutbox) MarkDelivered(_ context.Context, _ string) error { return nil }
func (s *stubOutbox) MarkFailed(_ context.Context, _, _ string, _ int) error {
	return nil
}

func mountTest(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	pkghttp.Mount(r, pkghttp.Dependencies{
		Packages:   &stubPackages{},
		Categories: &stubCategories{},
		Deliveries: &stubDeliveries{},
		Outbox:     &stubOutbox{},
	})
	return r
}

// TestDeliverManual_NoSignatureNoPhoto_400 verifica que un POST sin
// signature_url ni photo_evidence_url devuelve 400 + Problem JSON.
func TestDeliverManual_NoSignatureNoPhoto_400(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"guard_id": "11111111-2222-3333-4444-555555555555"
	}`)
	url := "/packages/22222222-3333-4444-5555-666666666666/deliver-manual"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("expected problem+json, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "signature_url") &&
		!strings.Contains(rec.Body.String(), "photo_evidence_url") {
		t.Errorf("expected body to mention required fields, got %s", rec.Body.String())
	}
}

// TestDeliverManual_OnlyPhoto_PassesValidation verifica que con SOLO
// photo_evidence_url el caso pasa la validacion (los stubs devuelven
// not-found en GetByID, asi que el handler responde 404; lo importante
// es que no es 400).
func TestDeliverManual_OnlyPhoto_PassesValidation(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"photo_evidence_url": "https://x/p.jpg",
		"guard_id": "11111111-2222-3333-4444-555555555555"
	}`)
	url := "/packages/22222222-3333-4444-5555-666666666666/deliver-manual"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusBadRequest {
		t.Fatalf("photo evidence should pass validation, got 400: %s", rec.Body.String())
	}
}

// TestListPackages_NoQuery_400 verifica que GET /packages sin params
// responde 400.
func TestListPackages_NoQuery_400(t *testing.T) {
	r := mountTest(t)
	req := httptest.NewRequest(http.MethodGet, "/packages", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestListCategories_OK verifica que GET /package-categories devuelve
// 200 con el sobre.
func TestListCategories_OK(t *testing.T) {
	r := mountTest(t)
	req := httptest.NewRequest(http.MethodGet, "/package-categories", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}
