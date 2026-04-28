package usecases_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/announcements/application/usecases"
	"github.com/saas-ph/api/internal/modules/announcements/domain"
	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

const (
	validUUID  = "11111111-2222-3333-4444-555555555555"
	otherUUID  = "22222222-3333-4444-5555-666666666666"
	thirdUUID  = "33333333-4444-5555-6666-777777777777"
	fourthUUID = "44444444-5555-6666-7777-888888888888"
)

// --- mocks ---

type fakeAnnouncementsRepo struct {
	createFn          func(ctx context.Context, in domain.CreateAnnouncementInput) (entities.Announcement, error)
	getByIDFn         func(ctx context.Context, id string) (entities.Announcement, error)
	archiveFn         func(ctx context.Context, id, actorID string) (entities.Announcement, error)
	listFeedForUserFn func(ctx context.Context, q domain.FeedQuery) ([]entities.Announcement, error)
}

func (f *fakeAnnouncementsRepo) Create(ctx context.Context, in domain.CreateAnnouncementInput) (entities.Announcement, error) {
	return f.createFn(ctx, in)
}
func (f *fakeAnnouncementsRepo) GetByID(ctx context.Context, id string) (entities.Announcement, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeAnnouncementsRepo) Archive(ctx context.Context, id, actorID string) (entities.Announcement, error) {
	return f.archiveFn(ctx, id, actorID)
}
func (f *fakeAnnouncementsRepo) ListFeedForUser(ctx context.Context, q domain.FeedQuery) ([]entities.Announcement, error) {
	return f.listFeedForUserFn(ctx, q)
}

type fakeAudiencesRepo struct {
	addFn     func(ctx context.Context, in domain.AddAudienceInput) (entities.Audience, error)
	listByIDF func(ctx context.Context, id string) ([]entities.Audience, error)
}

func (f *fakeAudiencesRepo) Add(ctx context.Context, in domain.AddAudienceInput) (entities.Audience, error) {
	return f.addFn(ctx, in)
}
func (f *fakeAudiencesRepo) ListByAnnouncement(ctx context.Context, id string) ([]entities.Audience, error) {
	return f.listByIDF(ctx, id)
}

type fakeAcksRepo struct {
	ackFn func(ctx context.Context, ann, user string) (entities.Ack, error)
}

func (f *fakeAcksRepo) Acknowledge(ctx context.Context, ann, user string) (entities.Ack, error) {
	return f.ackFn(ctx, ann, user)
}

// --- helpers ---

func mustProblem(t *testing.T, err error, status int) {
	t.Helper()
	var p apperrors.Problem
	if !errors.As(err, &p) {
		t.Fatalf("expected apperrors.Problem, got %v", err)
	}
	if p.Status != status {
		t.Fatalf("expected status %d, got %d (%s)", status, p.Status, p.Detail)
	}
}

// --- CreateAnnouncement ---

func TestCreateAnnouncement_Golden(t *testing.T) {
	now := time.Now()
	annRepo := &fakeAnnouncementsRepo{
		createFn: func(ctx context.Context, in domain.CreateAnnouncementInput) (entities.Announcement, error) {
			return entities.Announcement{
				ID:                validUUID,
				Title:             in.Title,
				Body:              in.Body,
				PublishedByUserID: in.PublishedByUserID,
				PublishedAt:       now,
				Pinned:            in.Pinned,
				Status:            entities.StatusPublished,
				CreatedAt:         now,
				UpdatedAt:         now,
				Version:           1,
			}, nil
		},
	}
	var addedAudiences []domain.AddAudienceInput
	audRepo := &fakeAudiencesRepo{
		addFn: func(ctx context.Context, in domain.AddAudienceInput) (entities.Audience, error) {
			addedAudiences = append(addedAudiences, in)
			return entities.Audience{
				ID:             otherUUID,
				AnnouncementID: in.AnnouncementID,
				TargetType:     in.TargetType,
				TargetID:       in.TargetID,
				Status:         "active",
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}

	uc := usecases.CreateAnnouncement{
		Announcements: annRepo,
		Audiences:     audRepo,
	}
	roleID := thirdUUID
	out, err := uc.Execute(context.Background(), usecases.CreateAnnouncementInput{
		Title:             "  Hello world  ",
		Body:              "Body of announcement",
		PublishedByUserID: validUUID,
		Pinned:            true,
		Audiences: []usecases.AudienceInput{
			{TargetType: "global"},
			{TargetType: "role", TargetID: &roleID},
		},
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Announcement.Title != "Hello world" {
		t.Errorf("expected title trimmed to %q, got %q", "Hello world", out.Announcement.Title)
	}
	if out.Announcement.Status != entities.StatusPublished {
		t.Errorf("expected status published, got %q", out.Announcement.Status)
	}
	if len(out.Audiences) != 2 {
		t.Fatalf("expected 2 audiences, got %d", len(out.Audiences))
	}
	if len(addedAudiences) != 2 {
		t.Errorf("expected Add called 2 times, got %d", len(addedAudiences))
	}
	if addedAudiences[0].TargetType != entities.TargetGlobal {
		t.Errorf("first audience type expected global, got %q", addedAudiences[0].TargetType)
	}
}

func TestCreateAnnouncement_ErrTitleRequired(t *testing.T) {
	uc := usecases.CreateAnnouncement{
		Announcements: &fakeAnnouncementsRepo{},
		Audiences:     &fakeAudiencesRepo{},
	}
	_, err := uc.Execute(context.Background(), usecases.CreateAnnouncementInput{
		Title:             "   ",
		Body:              "body",
		PublishedByUserID: validUUID,
		Audiences:         []usecases.AudienceInput{{TargetType: "global"}},
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateAnnouncement_ErrBodyRequired(t *testing.T) {
	uc := usecases.CreateAnnouncement{
		Announcements: &fakeAnnouncementsRepo{},
		Audiences:     &fakeAudiencesRepo{},
	}
	_, err := uc.Execute(context.Background(), usecases.CreateAnnouncementInput{
		Title:             "title",
		Body:              "",
		PublishedByUserID: validUUID,
		Audiences:         []usecases.AudienceInput{{TargetType: "global"}},
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateAnnouncement_ErrInvalidAudience_RoleMissingID(t *testing.T) {
	uc := usecases.CreateAnnouncement{
		Announcements: &fakeAnnouncementsRepo{},
		Audiences:     &fakeAudiencesRepo{},
	}
	_, err := uc.Execute(context.Background(), usecases.CreateAnnouncementInput{
		Title:             "title",
		Body:              "body",
		PublishedByUserID: validUUID,
		Audiences: []usecases.AudienceInput{
			{TargetType: "role", TargetID: nil},
		},
	})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestCreateAnnouncement_NoAudiences_BadRequest(t *testing.T) {
	uc := usecases.CreateAnnouncement{
		Announcements: &fakeAnnouncementsRepo{},
		Audiences:     &fakeAudiencesRepo{},
	}
	_, err := uc.Execute(context.Background(), usecases.CreateAnnouncementInput{
		Title:             "title",
		Body:              "body",
		PublishedByUserID: validUUID,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

// --- Acknowledge ---

func TestAcknowledge_Golden(t *testing.T) {
	now := time.Now()
	repo := &fakeAcksRepo{
		ackFn: func(ctx context.Context, ann, user string) (entities.Ack, error) {
			return entities.Ack{
				ID:             otherUUID,
				AnnouncementID: ann,
				UserID:         user,
				AcknowledgedAt: now,
				CreatedAt:      now,
				UpdatedAt:      now,
			}, nil
		},
	}
	uc := usecases.Acknowledge{Acks: repo}
	got, err := uc.Execute(context.Background(), usecases.AcknowledgeInput{
		AnnouncementID: validUUID,
		UserID:         thirdUUID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.AnnouncementID != validUUID || got.UserID != thirdUUID {
		t.Errorf("forwarding failed: %+v", got)
	}
}

func TestAcknowledge_BadAnnouncementID(t *testing.T) {
	uc := usecases.Acknowledge{Acks: &fakeAcksRepo{}}
	_, err := uc.Execute(context.Background(), usecases.AcknowledgeInput{
		AnnouncementID: "not-a-uuid",
		UserID:         thirdUUID,
	})
	mustProblem(t, err, http.StatusBadRequest)
}

// --- ListFeed ---

func TestListFeed_Golden(t *testing.T) {
	now := time.Now()
	want := []entities.Announcement{
		{ID: validUUID, Title: "Pinned", Body: "x", Status: entities.StatusPublished, Pinned: true, PublishedAt: now, CreatedAt: now, UpdatedAt: now, Version: 1},
		{ID: otherUUID, Title: "Recent", Body: "y", Status: entities.StatusPublished, PublishedAt: now.Add(-time.Hour), CreatedAt: now, UpdatedAt: now, Version: 1},
	}
	var captured domain.FeedQuery
	repo := &fakeAnnouncementsRepo{
		listFeedForUserFn: func(ctx context.Context, q domain.FeedQuery) ([]entities.Announcement, error) {
			captured = q
			return want, nil
		},
	}
	uc := usecases.ListFeed{Announcements: repo}
	out, err := uc.Execute(context.Background(), usecases.FeedInput{
		UserID:       validUUID,
		RoleIDs:      []string{thirdUUID},
		StructureIDs: nil,
		UnitIDs:      []string{fourthUUID},
		// Limit/Offset por defecto.
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out.Items))
	}
	if out.Total != 2 {
		t.Errorf("expected total=2, got %d", out.Total)
	}
	if captured.Limit != 20 {
		t.Errorf("expected default limit=20, got %d", captured.Limit)
	}
	if captured.UserID != validUUID {
		t.Errorf("expected userID forwarded, got %q", captured.UserID)
	}
	if len(captured.RoleIDs) != 1 || captured.RoleIDs[0] != thirdUUID {
		t.Errorf("role_ids not forwarded: %+v", captured.RoleIDs)
	}
}

// --- ArchiveAnnouncement ---

func TestArchiveAnnouncement_NotFound(t *testing.T) {
	repo := &fakeAnnouncementsRepo{
		archiveFn: func(ctx context.Context, id, actorID string) (entities.Announcement, error) {
			return entities.Announcement{}, domain.ErrAnnouncementNotFound
		},
	}
	uc := usecases.ArchiveAnnouncement{Announcements: repo}
	_, err := uc.Execute(context.Background(), usecases.ArchiveAnnouncementInput{
		ID: validUUID,
	})
	mustProblem(t, err, http.StatusNotFound)
}

func TestArchiveAnnouncement_BadID(t *testing.T) {
	uc := usecases.ArchiveAnnouncement{Announcements: &fakeAnnouncementsRepo{}}
	_, err := uc.Execute(context.Background(), usecases.ArchiveAnnouncementInput{
		ID: "x",
	})
	mustProblem(t, err, http.StatusBadRequest)
}

// --- GetAnnouncement ---

func TestGetAnnouncement_NotFound(t *testing.T) {
	repo := &fakeAnnouncementsRepo{
		getByIDFn: func(ctx context.Context, id string) (entities.Announcement, error) {
			return entities.Announcement{}, domain.ErrAnnouncementNotFound
		},
	}
	uc := usecases.GetAnnouncement{Announcements: repo, Audiences: &fakeAudiencesRepo{}}
	_, err := uc.Execute(context.Background(), validUUID)
	mustProblem(t, err, http.StatusNotFound)
}

func TestGetAnnouncement_OK(t *testing.T) {
	now := time.Now()
	annRepo := &fakeAnnouncementsRepo{
		getByIDFn: func(ctx context.Context, id string) (entities.Announcement, error) {
			return entities.Announcement{ID: id, Title: "t", Body: "b", Status: entities.StatusPublished, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	audRepo := &fakeAudiencesRepo{
		listByIDF: func(ctx context.Context, id string) ([]entities.Audience, error) {
			return []entities.Audience{{ID: otherUUID, AnnouncementID: id, TargetType: entities.TargetGlobal}}, nil
		},
	}
	uc := usecases.GetAnnouncement{Announcements: annRepo, Audiences: audRepo}
	out, err := uc.Execute(context.Background(), validUUID)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Announcement.ID != validUUID {
		t.Errorf("forwarding failed: %+v", out.Announcement)
	}
	if len(out.Audiences) != 1 {
		t.Errorf("expected 1 audience, got %d", len(out.Audiences))
	}
}
