package usecases_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/tenant_config/application/usecases"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// --- mocks ---

type fakeSettingsRepo struct {
	listFn    func(ctx context.Context, f domain.ListSettingsFilter) ([]entities.Setting, int64, error)
	getFn     func(ctx context.Context, key string) (entities.Setting, error)
	upsertFn  func(ctx context.Context, in domain.UpsertSettingInput) (entities.Setting, error)
	archiveFn func(ctx context.Context, key, actor string) (entities.Setting, error)
}

func (f *fakeSettingsRepo) List(ctx context.Context, fl domain.ListSettingsFilter) ([]entities.Setting, int64, error) {
	return f.listFn(ctx, fl)
}
func (f *fakeSettingsRepo) Get(ctx context.Context, key string) (entities.Setting, error) {
	return f.getFn(ctx, key)
}
func (f *fakeSettingsRepo) Upsert(ctx context.Context, in domain.UpsertSettingInput) (entities.Setting, error) {
	return f.upsertFn(ctx, in)
}
func (f *fakeSettingsRepo) Archive(ctx context.Context, key, actorID string) (entities.Setting, error) {
	return f.archiveFn(ctx, key, actorID)
}

type fakeBrandingRepo struct {
	getFn    func(ctx context.Context) (entities.Branding, error)
	updateFn func(ctx context.Context, in domain.UpdateBrandingInput) (entities.Branding, error)
}

func (f *fakeBrandingRepo) Get(ctx context.Context) (entities.Branding, error) {
	return f.getFn(ctx)
}
func (f *fakeBrandingRepo) Update(ctx context.Context, in domain.UpdateBrandingInput) (entities.Branding, error) {
	return f.updateFn(ctx, in)
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

// --- GetSetting ---

func TestGetSetting_BadKey(t *testing.T) {
	uc := usecases.GetSetting{Repo: &fakeSettingsRepo{}}
	_, err := uc.Execute(context.Background(), "Invalid Key!")
	mustProblem(t, err, http.StatusBadRequest)
}

func TestGetSetting_NotFound(t *testing.T) {
	uc := usecases.GetSetting{Repo: &fakeSettingsRepo{
		getFn: func(ctx context.Context, key string) (entities.Setting, error) {
			return entities.Setting{}, domain.ErrSettingNotFound
		},
	}}
	_, err := uc.Execute(context.Background(), "contact.email")
	mustProblem(t, err, http.StatusNotFound)
}

func TestGetSetting_OK(t *testing.T) {
	want := entities.Setting{Key: "contact.email", Value: []byte(`"x@y"`), Status: entities.SettingStatusActive, Version: 1}
	uc := usecases.GetSetting{Repo: &fakeSettingsRepo{
		getFn: func(ctx context.Context, key string) (entities.Setting, error) { return want, nil },
	}}
	got, err := uc.Execute(context.Background(), "contact.email")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.Key != want.Key {
		t.Fatalf("got %q want %q", got.Key, want.Key)
	}
}

func TestGetSetting_RepoError(t *testing.T) {
	uc := usecases.GetSetting{Repo: &fakeSettingsRepo{
		getFn: func(ctx context.Context, key string) (entities.Setting, error) {
			return entities.Setting{}, errors.New("boom")
		},
	}}
	_, err := uc.Execute(context.Background(), "contact.email")
	mustProblem(t, err, http.StatusInternalServerError)
}

// --- ListSettings ---

func TestListSettings_DefaultsAndForwarding(t *testing.T) {
	var captured domain.ListSettingsFilter
	uc := usecases.ListSettings{Repo: &fakeSettingsRepo{
		listFn: func(ctx context.Context, f domain.ListSettingsFilter) ([]entities.Setting, int64, error) {
			captured = f
			return []entities.Setting{}, 0, nil
		},
	}}
	out, err := uc.Execute(context.Background(), usecases.ListInput{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if captured.Limit != 50 {
		t.Errorf("default limit not applied: got %d", captured.Limit)
	}
	if out.Limit != 50 {
		t.Errorf("output limit: got %d", out.Limit)
	}
}

func TestListSettings_LimitClamped(t *testing.T) {
	var captured domain.ListSettingsFilter
	uc := usecases.ListSettings{Repo: &fakeSettingsRepo{
		listFn: func(ctx context.Context, f domain.ListSettingsFilter) ([]entities.Setting, int64, error) {
			captured = f
			return []entities.Setting{}, 0, nil
		},
	}}
	if _, err := uc.Execute(context.Background(), usecases.ListInput{Limit: 9999, Offset: -10}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if captured.Limit != 200 {
		t.Errorf("expected limit clamp to 200, got %d", captured.Limit)
	}
	if captured.Offset != 0 {
		t.Errorf("expected offset 0, got %d", captured.Offset)
	}
}

// --- SetSetting ---

func TestSetSetting_BadKey(t *testing.T) {
	uc := usecases.SetSetting{Repo: &fakeSettingsRepo{}}
	_, err := uc.Execute(context.Background(), usecases.SetInput{Key: "BAD KEY", Value: []byte("true")})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestSetSetting_EmptyValue(t *testing.T) {
	uc := usecases.SetSetting{Repo: &fakeSettingsRepo{}}
	_, err := uc.Execute(context.Background(), usecases.SetInput{Key: "x.y", Value: nil})
	mustProblem(t, err, http.StatusBadRequest)
}

func TestSetSetting_OK(t *testing.T) {
	uc := usecases.SetSetting{Repo: &fakeSettingsRepo{
		upsertFn: func(ctx context.Context, in domain.UpsertSettingInput) (entities.Setting, error) {
			return entities.Setting{Key: in.Key, Value: in.Value, Status: entities.SettingStatusActive, Version: 2}, nil
		},
	}}
	got, err := uc.Execute(context.Background(), usecases.SetInput{Key: "x.y", Value: []byte("true")})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.Version != 2 {
		t.Fatalf("got version %d", got.Version)
	}
}

// --- ArchiveSetting ---

func TestArchiveSetting_NotFound(t *testing.T) {
	uc := usecases.ArchiveSetting{Repo: &fakeSettingsRepo{
		archiveFn: func(ctx context.Context, key, actor string) (entities.Setting, error) {
			return entities.Setting{}, domain.ErrSettingNotFound
		},
	}}
	_, err := uc.Execute(context.Background(), usecases.ArchiveInput{Key: "contact.email"})
	mustProblem(t, err, http.StatusNotFound)
}

// --- GetBranding ---

func TestGetBranding_NotFound(t *testing.T) {
	uc := usecases.GetBranding{Repo: &fakeBrandingRepo{
		getFn: func(ctx context.Context) (entities.Branding, error) {
			return entities.Branding{}, domain.ErrBrandingNotFound
		},
	}}
	_, err := uc.Execute(context.Background())
	mustProblem(t, err, http.StatusNotFound)
}

func TestGetBranding_OK(t *testing.T) {
	uc := usecases.GetBranding{Repo: &fakeBrandingRepo{
		getFn: func(ctx context.Context) (entities.Branding, error) {
			return entities.Branding{
				DisplayName: "Test", Timezone: "America/Bogota", Locale: "es-CO",
				Status: entities.BrandingStatusActive, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}}
	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.DisplayName != "Test" {
		t.Fatalf("got %q", got.DisplayName)
	}
}

// --- UpdateBranding ---

func TestUpdateBranding_ValidationFails(t *testing.T) {
	uc := usecases.UpdateBranding{Repo: &fakeBrandingRepo{}}
	cases := []struct {
		name string
		in   usecases.UpdateBrandingInput
	}{
		{"empty_display_name", usecases.UpdateBrandingInput{DisplayName: "", Timezone: "America/Bogota", Locale: "es-CO", ExpectedVersion: 1}},
		{"bad_color", usecases.UpdateBrandingInput{DisplayName: "X", PrimaryColor: ptr("not-a-color"), Timezone: "America/Bogota", Locale: "es-CO", ExpectedVersion: 1}},
		{"bad_tz", usecases.UpdateBrandingInput{DisplayName: "X", Timezone: "Europe/Madrid", Locale: "es-CO", ExpectedVersion: 1}},
		{"bad_locale", usecases.UpdateBrandingInput{DisplayName: "X", Timezone: "UTC", Locale: "INVALID", ExpectedVersion: 1}},
		{"missing_version", usecases.UpdateBrandingInput{DisplayName: "X", Timezone: "UTC", Locale: "es-CO", ExpectedVersion: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), tc.in)
			mustProblem(t, err, http.StatusBadRequest)
		})
	}
}

func TestUpdateBranding_VersionMismatch(t *testing.T) {
	uc := usecases.UpdateBranding{Repo: &fakeBrandingRepo{
		updateFn: func(ctx context.Context, in domain.UpdateBrandingInput) (entities.Branding, error) {
			return entities.Branding{}, domain.ErrVersionMismatch
		},
	}}
	_, err := uc.Execute(context.Background(), usecases.UpdateBrandingInput{
		DisplayName: "X", Timezone: "UTC", Locale: "es-CO", ExpectedVersion: 1,
	})
	mustProblem(t, err, http.StatusConflict)
}

func TestUpdateBranding_OK(t *testing.T) {
	called := false
	uc := usecases.UpdateBranding{Repo: &fakeBrandingRepo{
		updateFn: func(ctx context.Context, in domain.UpdateBrandingInput) (entities.Branding, error) {
			called = true
			if in.DisplayName != "Acacias" {
				t.Errorf("display_name forwarding failed: %q", in.DisplayName)
			}
			return entities.Branding{
				DisplayName: in.DisplayName, Timezone: in.Timezone, Locale: in.Locale,
				Status: entities.BrandingStatusActive, Version: in.ExpectedVersion + 1,
			}, nil
		},
	}}
	got, err := uc.Execute(context.Background(), usecases.UpdateBrandingInput{
		DisplayName: "Acacias", PrimaryColor: ptr("#aabbcc"), Timezone: "America/Bogota", Locale: "es-CO", ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !called {
		t.Fatal("repo not called")
	}
	if got.Version != 2 {
		t.Fatalf("version: %d", got.Version)
	}
}

func ptr(s string) *string { return &s }
