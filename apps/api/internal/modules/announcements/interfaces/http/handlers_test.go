package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/announcements/domain"
	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
	annhttp "github.com/saas-ph/api/internal/modules/announcements/interfaces/http"
)

// --- in-memory fakes (minimo necesario para el test) ---

type stubAnnouncements struct{}

func (s *stubAnnouncements) Create(_ context.Context, _ domain.CreateAnnouncementInput) (entities.Announcement, error) {
	return entities.Announcement{}, nil
}
func (s *stubAnnouncements) GetByID(_ context.Context, _ string) (entities.Announcement, error) {
	return entities.Announcement{}, domain.ErrAnnouncementNotFound
}
func (s *stubAnnouncements) Archive(_ context.Context, _, _ string) (entities.Announcement, error) {
	return entities.Announcement{}, domain.ErrAnnouncementNotFound
}
func (s *stubAnnouncements) ListFeedForUser(_ context.Context, _ domain.FeedQuery) ([]entities.Announcement, error) {
	return nil, nil
}

type stubAudiences struct{}

func (s *stubAudiences) Add(_ context.Context, _ domain.AddAudienceInput) (entities.Audience, error) {
	return entities.Audience{}, nil
}
func (s *stubAudiences) ListByAnnouncement(_ context.Context, _ string) ([]entities.Audience, error) {
	return nil, nil
}

type stubAcks struct{}

func (s *stubAcks) Acknowledge(_ context.Context, _, _ string) (entities.Ack, error) {
	return entities.Ack{}, nil
}

func mountTest(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	annhttp.Mount(r, annhttp.Dependencies{
		AnnouncementsRepo: &stubAnnouncements{},
		AudiencesRepo:     &stubAudiences{},
		AcksRepo:          &stubAcks{},
	})
	return r
}

// TestCreateAnnouncement_NoTitle_400 verifica que un POST sin title
// devuelve 400 + Problem JSON.
func TestCreateAnnouncement_NoTitle_400(t *testing.T) {
	r := mountTest(t)
	body := []byte(`{
		"title": "",
		"body": "some body",
		"published_by_user_id": "11111111-2222-3333-4444-555555555555",
		"audiences": [{"target_type": "global"}]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/announcements", bytes.NewReader(body))
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
	if !strings.Contains(rec.Body.String(), "title") {
		t.Errorf("expected body to mention title, got %s", rec.Body.String())
	}
}
