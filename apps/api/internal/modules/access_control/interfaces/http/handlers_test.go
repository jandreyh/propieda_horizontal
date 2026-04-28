package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	achttp "github.com/saas-ph/api/internal/modules/access_control/interfaces/http"
)

// --- in-memory fakes (minimo necesario para el test) ---

type stubBlacklist struct{}

func (s *stubBlacklist) Get(ctx context.Context, _ entities.DocumentType, _ string) (*entities.BlacklistEntry, error) {
	return nil, nil
}
func (s *stubBlacklist) Create(_ context.Context, _ domain.CreateBlacklistInput) (entities.BlacklistEntry, error) {
	return entities.BlacklistEntry{}, nil
}
func (s *stubBlacklist) List(_ context.Context) ([]entities.BlacklistEntry, error) { return nil, nil }
func (s *stubBlacklist) Archive(_ context.Context, _, _ string) (entities.BlacklistEntry, error) {
	return entities.BlacklistEntry{}, nil
}

type stubPreReg struct{}

func (s *stubPreReg) Create(_ context.Context, _ domain.CreatePreRegistrationInput) (entities.PreRegistration, error) {
	return entities.PreRegistration{}, nil
}
func (s *stubPreReg) GetByQRHash(_ context.Context, _ string) (entities.PreRegistration, error) {
	return entities.PreRegistration{}, domain.ErrPreregistrationNotFound
}
func (s *stubPreReg) ConsumeOne(_ context.Context, _ string) (entities.PreRegistration, error) {
	return entities.PreRegistration{}, domain.ErrPreregistrationNotFound
}

type stubEntries struct{}

func (s *stubEntries) Create(_ context.Context, _ domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
	return entities.VisitorEntry{}, nil
}
func (s *stubEntries) Close(_ context.Context, _, _ string) (entities.VisitorEntry, error) {
	return entities.VisitorEntry{}, nil
}
func (s *stubEntries) ListActive(_ context.Context) ([]entities.VisitorEntry, error) {
	return nil, nil
}
func (s *stubEntries) GetByID(_ context.Context, _ string) (entities.VisitorEntry, error) {
	return entities.VisitorEntry{}, domain.ErrEntryNotFound
}

func mountTest(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	achttp.Mount(r, achttp.Dependencies{
		BlacklistRepo: &stubBlacklist{},
		PreRegRepo:    &stubPreReg{},
		EntryRepo:     &stubEntries{},
	})
	return r
}

// TestCheckinManual_NoPhoto_400 verifica que un POST sin photo_url
// devuelve 400 + Problem JSON.
func TestCheckinManual_NoPhoto_400(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"visitor_full_name": "Juan",
		"visitor_document_type": "CC",
		"visitor_document_number": "12345",
		"photo_url": "",
		"guard_id": "11111111-2222-3333-4444-555555555555"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/visits/checkin-manual", bytes.NewReader(body))
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
	if !strings.Contains(rec.Body.String(), "photo_url") {
		t.Errorf("expected body to mention photo_url, got %s", rec.Body.String())
	}
}
