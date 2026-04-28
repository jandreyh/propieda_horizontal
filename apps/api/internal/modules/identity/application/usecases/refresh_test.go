package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/application/usecases"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
)

func TestRefreshUseCase_GoldenPath(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	users.put(user)

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	plain := "rt-123"
	hash := usecases.HashRefreshToken(plain)
	sessions.byID["sess-1"] = &entities.Session{
		ID:        "sess-1",
		UserID:    user.ID,
		TokenHash: hash,
		IssuedAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
		Status:    entities.SessionStatusActive,
	}
	sessions.byTokenHash[hash] = sessions.byID["sess-1"]

	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      func() time.Time { return now },
	})
	resp, err := uc.Execute(newTenantCtx(), dto.RefreshRequest{RefreshToken: plain})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", resp)
	}
	// la sesion vieja debe quedar revocada con motivo rotated
	old := sessions.byID["sess-1"]
	if old.RevokedAt == nil {
		t.Fatalf("expected old session revoked")
	}
	if old.RevocationReasonValue() != entities.RevocationReasonRotated {
		t.Fatalf("expected reason=rotated, got %s", old.RevocationReasonValue())
	}
	// debe haber 2 sesiones en total
	if len(sessions.byID) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions.byID))
	}
}

func TestRefreshUseCase_EmptyToken(t *testing.T) {
	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
	})
	_, err := uc.Execute(newTenantCtx(), dto.RefreshRequest{RefreshToken: ""})
	if !errors.Is(err, usecases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRefreshUseCase_UnknownToken(t *testing.T) {
	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
	})
	_, err := uc.Execute(newTenantCtx(), dto.RefreshRequest{RefreshToken: "ghost"})
	if !errors.Is(err, usecases.ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}

func TestRefreshUseCase_ReuseDetected(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	users.put(user)

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	plain := "rt-rot"
	hash := usecases.HashRefreshToken(plain)
	revAt := now.Add(-time.Minute)
	reason := entities.RevocationReasonRotated
	sessions.byID["sess-1"] = &entities.Session{
		ID:               "sess-1",
		UserID:           user.ID,
		TokenHash:        hash,
		IssuedAt:         now.Add(-time.Hour),
		ExpiresAt:        now.Add(24 * time.Hour),
		Status:           entities.SessionStatusRevoked,
		RevokedAt:        &revAt,
		RevocationReason: &reason,
	}
	sessions.byTokenHash[hash] = sessions.byID["sess-1"]
	// crear hijo "sess-2" para verificar que la cadena cae
	sessions.byID["sess-2"] = &entities.Session{
		ID:              "sess-2",
		UserID:          user.ID,
		TokenHash:       "child-hash",
		IssuedAt:        now,
		ExpiresAt:       now.Add(24 * time.Hour),
		ParentSessionID: ptrString("sess-1"),
		Status:          entities.SessionStatusActive,
	}
	sessions.byTokenHash["child-hash"] = sessions.byID["sess-2"]

	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      func() time.Time { return now },
	})
	_, err := uc.Execute(newTenantCtx(), dto.RefreshRequest{RefreshToken: plain})
	if !errors.Is(err, usecases.ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
	// la cadena entera debe estar marcada con motivo reuse_detected
	child := sessions.byID["sess-2"]
	if child.RevokedAt == nil {
		t.Fatalf("expected child revoked")
	}
	if child.RevocationReasonValue() != entities.RevocationReasonReuseDetected {
		t.Fatalf("expected reason=reuse_detected, got %s", child.RevocationReasonValue())
	}
}

func TestRefreshUseCase_ExpiredSession(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	users.put(user)

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	plain := "rt-exp"
	hash := usecases.HashRefreshToken(plain)
	sessions.byID["sess-exp"] = &entities.Session{
		ID:        "sess-exp",
		UserID:    user.ID,
		TokenHash: hash,
		IssuedAt:  now.Add(-30 * 24 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
		Status:    entities.SessionStatusActive,
	}
	sessions.byTokenHash[hash] = sessions.byID["sess-exp"]

	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      func() time.Time { return now },
	})
	_, err := uc.Execute(newTenantCtx(), dto.RefreshRequest{RefreshToken: plain})
	if !errors.Is(err, usecases.ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}

func TestRefreshUseCase_InactiveUser(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	user.Status = entities.UserStatusSuspended
	users.put(user)

	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	plain := "rt-x"
	hash := usecases.HashRefreshToken(plain)
	sessions.byID["sess-1"] = &entities.Session{
		ID:        "sess-1",
		UserID:    user.ID,
		TokenHash: hash,
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour),
		Status:    entities.SessionStatusActive,
	}
	sessions.byTokenHash[hash] = sessions.byID["sess-1"]

	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      func() time.Time { return now },
	})
	_, err := uc.Execute(newTenantCtx(), dto.RefreshRequest{RefreshToken: plain})
	if !errors.Is(err, usecases.ErrAccountInactive) {
		t.Fatalf("expected ErrAccountInactive, got %v", err)
	}
}

func TestRefreshUseCase_MissingTenant(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	users.put(user)
	plain := "rt-z"
	hash := usecases.HashRefreshToken(plain)
	now := time.Now()
	sessions.byID["sess-1"] = &entities.Session{
		ID:        "sess-1",
		UserID:    user.ID,
		TokenHash: hash,
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour),
		Status:    entities.SessionStatusActive,
	}
	sessions.byTokenHash[hash] = sessions.byID["sess-1"]
	uc := usecases.NewRefreshUseCase(usecases.RefreshDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
	})
	_, err := uc.Execute(context.Background(), dto.RefreshRequest{RefreshToken: plain})
	if err == nil {
		t.Fatalf("expected error when tenant missing")
	}
}
