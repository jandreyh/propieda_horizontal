package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/application/usecases"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/passwords"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

func newSigner(t *testing.T) *jwtsign.Signer {
	t.Helper()
	s, err := jwtsign.NewSigner(jwtsign.SignerConfig{Issuer: "test", Audience: "test"})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return s
}

func newTenantCtx() context.Context {
	return tenantctx.WithTenant(context.Background(), &tenantctx.Tenant{
		ID:          "tenant-123",
		Slug:        "demo",
		DisplayName: "Demo",
		Pool:        (*pgxpool.Pool)(nil),
	})
}

func ptrString(s string) *string { return &s }

func mustHash(t *testing.T, plain string) string {
	t.Helper()
	h, err := passwords.Hash(plain)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return h
}

func newActiveUser(t *testing.T, password string) *entities.User {
	t.Helper()
	email := "u@example.com"
	return &entities.User{
		ID:             "user-1",
		DocumentType:   entities.DocumentTypeCC,
		DocumentNumber: "12345",
		Names:          "Ana",
		LastNames:      "Perez",
		Email:          &email,
		PasswordHash:   mustHash(t, password),
		Status:         entities.UserStatusActive,
	}
}

func TestLoginUseCase_GoldenPathNoMFA(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	users.put(user)

	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      func() time.Time { return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC) },
	})

	resp, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "u@example.com", Password: "S3cret!"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.MFARequired {
		t.Fatalf("expected MFARequired=false")
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", resp)
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("unexpected token_type: %s", resp.TokenType)
	}
	if resp.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", resp.ExpiresIn)
	}
	// debe haber 1 sesion creada y nada en revoke
	if len(sessions.byID) != 1 {
		t.Fatalf("expected 1 session created, got %d", len(sessions.byID))
	}
	if users.lastLoginAt.IsZero() {
		t.Fatalf("expected last_login_at to be updated")
	}
}

func TestLoginUseCase_DocumentIdentifier(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	user.Email = nil // sin email
	users.put(user)

	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      time.Now,
	})
	resp, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "CC:12345", Password: "S3cret!"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatalf("expected access token")
	}
}

func TestLoginUseCase_GoldenPathWithMFA(t *testing.T) {
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	user := newActiveUser(t, "S3cret!")
	user.MFASecret = ptrString("JBSWY3DPEHPK3PXP")
	users.put(user)

	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   newSigner(t),
		Now:      time.Now,
	})
	resp, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "u@example.com", Password: "S3cret!"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !resp.MFARequired {
		t.Fatalf("expected MFARequired=true")
	}
	if resp.PreAuthToken == "" {
		t.Fatalf("expected pre_auth_token, got %+v", resp)
	}
	if resp.AccessToken != "" || resp.RefreshToken != "" {
		t.Fatalf("should not issue session tokens with pending MFA")
	}
	if len(sessions.byID) != 0 {
		t.Fatalf("no session should be created with pending MFA")
	}
}

func TestLoginUseCase_InvalidInput(t *testing.T) {
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
	})
	cases := []dto.LoginRequest{
		{Identifier: "", Password: "x"},
		{Identifier: "u@example.com", Password: ""},
		{Identifier: "   ", Password: "x"},
	}
	for _, tc := range cases {
		_, err := uc.Execute(newTenantCtx(), tc)
		if !errors.Is(err, usecases.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for %+v, got %v", tc, err)
		}
	}
}

func TestLoginUseCase_UnknownUser(t *testing.T) {
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
	})
	_, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "ghost@example.com", Password: "x"})
	if !errors.Is(err, usecases.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestLoginUseCase_LockedUser(t *testing.T) {
	users := newUserRepoMock()
	user := newActiveUser(t, "S3cret!")
	until := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	user.LockedUntil = &until
	users.put(user)
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
		Now:      func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	})
	_, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "u@example.com", Password: "S3cret!"})
	if !errors.Is(err, usecases.ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked, got %v", err)
	}
}

func TestLoginUseCase_BadPasswordIncrementsCounter(t *testing.T) {
	users := newUserRepoMock()
	user := newActiveUser(t, "S3cret!")
	user.FailedLoginAttempts = 2
	users.put(user)
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
		Now:      time.Now,
	})
	_, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "u@example.com", Password: "wrong!"})
	if !errors.Is(err, usecases.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if users.failedAttemptsCalls != 1 {
		t.Fatalf("expected IncrementFailedAttempts called once, got %d", users.failedAttemptsCalls)
	}
	if users.lockCalls != 0 {
		t.Fatalf("should not lock yet")
	}
}

func TestLoginUseCase_BadPasswordHitsLockout(t *testing.T) {
	users := newUserRepoMock()
	user := newActiveUser(t, "S3cret!")
	user.FailedLoginAttempts = 4 // next attempt = 5 -> lockout
	users.put(user)
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
		Now:      func() time.Time { return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC) },
	})
	_, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "u@example.com", Password: "wrong!"})
	if !errors.Is(err, usecases.ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked, got %v", err)
	}
	if users.lockCalls != 1 {
		t.Fatalf("expected LockUser called once, got %d", users.lockCalls)
	}
	wantUntil := time.Date(2026, 4, 28, 12, 15, 0, 0, time.UTC)
	if !users.lastLockUntil.Equal(wantUntil) {
		t.Fatalf("expected lockUntil=%s got %s", wantUntil, users.lastLockUntil)
	}
}

func TestLoginUseCase_MissingTenant(t *testing.T) {
	users := newUserRepoMock()
	user := newActiveUser(t, "S3cret!")
	users.put(user)
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
		Now:      time.Now,
	})
	_, err := uc.Execute(context.Background(), dto.LoginRequest{Identifier: "u@example.com", Password: "S3cret!"})
	if err == nil {
		t.Fatalf("expected error when tenant missing")
	}
	if !strings.Contains(err.Error(), "tenant") {
		t.Fatalf("expected tenant-related error, got: %v", err)
	}
}

func TestLoginUseCase_InactiveUser(t *testing.T) {
	users := newUserRepoMock()
	user := newActiveUser(t, "S3cret!")
	user.Status = entities.UserStatusInactive
	users.put(user)
	uc := usecases.NewLoginUseCase(usecases.LoginDeps{
		Users:    users,
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
		Now:      time.Now,
	})
	_, err := uc.Execute(newTenantCtx(), dto.LoginRequest{Identifier: "u@example.com", Password: "S3cret!"})
	if !errors.Is(err, usecases.ErrAccountInactive) {
		t.Fatalf("expected ErrAccountInactive, got %v", err)
	}
}
