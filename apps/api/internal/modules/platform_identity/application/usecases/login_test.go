package usecases

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/passwords"
)

// fakeRepo implementa domain.PlatformUserRepository en memoria.
type fakeRepo struct {
	byEmail        map[string]*entities.PlatformUser
	memberships    []entities.Membership
	memberErr      error
	markErr        error
	incrementErr   error
	incrementCalls int
}

func (f *fakeRepo) FindByEmail(_ context.Context, email string) (*entities.PlatformUser, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeRepo) FindByDocument(context.Context, string, string) (*entities.PlatformUser, error) {
	return nil, domain.ErrUserNotFound
}

func (f *fakeRepo) FindByID(context.Context, uuid.UUID) (*entities.PlatformUser, error) {
	return nil, domain.ErrUserNotFound
}

func (f *fakeRepo) FindByPublicCode(context.Context, string) (*entities.PlatformUser, error) {
	return nil, domain.ErrUserNotFound
}

func (f *fakeRepo) MarkLoginSuccess(context.Context, uuid.UUID, time.Time) error {
	return f.markErr
}

func (f *fakeRepo) IncrementFailedLogin(context.Context, uuid.UUID) (int32, *time.Time, error) {
	f.incrementCalls++
	return int32(f.incrementCalls), nil, f.incrementErr
}

func (f *fakeRepo) ListMemberships(context.Context, uuid.UUID) ([]entities.Membership, error) {
	return f.memberships, f.memberErr
}

func (f *fakeRepo) HasMembership(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}

// helper builds a signer with an ephemeral ed25519 key.
func newTestSigner(t *testing.T) *jwtsign.Signer {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	signer, err := jwtsign.NewSigner(jwtsign.SignerConfig{
		KeyID:      "test-key",
		PrivateKey: priv,
		PublicKey:  pub,
		Issuer:     "test",
		Audience:   "test",
	})
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer
}

func newActiveUser(t *testing.T, email, password string, mfaEnrolled bool) *entities.PlatformUser {
	t.Helper()
	hash, err := passwords.Hash(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &entities.PlatformUser{
		ID:             uuid.New(),
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Names:          "Ana",
		LastNames:      "Demo",
		Email:          email,
		PasswordHash:   hash,
		PublicCode:     "ABCD-EFGH-JKLM",
		Status:         "active",
	}
	if mfaEnrolled {
		now := time.Now().UTC()
		u.MFAEnrolledAt = &now
	}
	return u
}

func TestLogin_Success_NoMFA_OneMembership(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	tenantID := uuid.New()
	repo := &fakeRepo{
		byEmail: map[string]*entities.PlatformUser{user.Email: user},
		memberships: []entities.Membership{{
			TenantID:   tenantID,
			TenantSlug: "demo",
			TenantName: "Conjunto Demo",
			Role:       "tenant_admin",
			Status:     "active",
		}},
	}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	res, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          "Ana@Demo.Test",
		DocumentType:   "cc",
		DocumentNumber: "1000000001",
		Password:       "secret123",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.AccessToken == "" {
		t.Error("expected access_token")
	}
	if res.MFARequired || res.PreAuthToken != "" {
		t.Error("did not expect MFA required")
	}
	if len(res.Memberships) != 1 || res.Memberships[0].TenantSlug != "demo" {
		t.Errorf("memberships = %+v", res.Memberships)
	}
	if res.NeedsTenant {
		t.Error("with 1 membership, NeedsTenant should be false")
	}
	if res.TokenType != "Bearer" {
		t.Errorf("token_type = %q", res.TokenType)
	}
}

func TestLogin_Success_MFAEnrolled_ReturnsPreAuth(t *testing.T) {
	user := newActiveUser(t, "mfa@demo.test", "secret123", true)
	repo := &fakeRepo{
		byEmail: map[string]*entities.PlatformUser{user.Email: user},
	}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	res, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Password:       "secret123",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !res.MFARequired || res.PreAuthToken == "" {
		t.Errorf("expected MFA pre-auth, got %+v", res)
	}
	if res.AccessToken != "" {
		t.Error("must not return access_token when MFA pending")
	}
}

func TestLogin_InvalidEmail(t *testing.T) {
	repo := &fakeRepo{byEmail: map[string]*entities.PlatformUser{}}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          "ghost@demo.test",
		DocumentType:   "CC",
		DocumentNumber: "1",
		Password:       "x",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_DocumentMismatch(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	repo := &fakeRepo{byEmail: map[string]*entities.PlatformUser{user.Email: user}}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "2222222222",
		Password:       "secret123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_BadPassword_Increments(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	repo := &fakeRepo{byEmail: map[string]*entities.PlatformUser{user.Email: user}}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Password:       "wrong",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if repo.incrementCalls != 1 {
		t.Errorf("expected 1 increment call, got %d", repo.incrementCalls)
	}
}

func TestLogin_AccountInactive(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	user.Status = "suspended"
	repo := &fakeRepo{byEmail: map[string]*entities.PlatformUser{user.Email: user}}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Password:       "secret123",
	})
	if !errors.Is(err, ErrAccountInactive) {
		t.Fatalf("expected ErrAccountInactive, got %v", err)
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	future := time.Now().Add(10 * time.Minute)
	user.LockedUntil = &future
	repo := &fakeRepo{byEmail: map[string]*entities.PlatformUser{user.Email: user}}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Password:       "secret123",
	})
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked, got %v", err)
	}
}

func TestLogin_InvalidInput(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email: "x@y", DocumentType: "", DocumentNumber: "", Password: "",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLogin_MultipleMemberships_NeedsTenant(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	repo := &fakeRepo{
		byEmail: map[string]*entities.PlatformUser{user.Email: user},
		memberships: []entities.Membership{
			{TenantID: uuid.New(), TenantSlug: "demo", TenantName: "Demo", Role: "tenant_admin", Status: "active"},
			{TenantID: uuid.New(), TenantSlug: "demo2", TenantName: "Demo 2", Role: "guard", Status: "active"},
		},
	}
	uc := NewLoginUseCase(LoginDeps{Users: repo, Signer: newTestSigner(t)})

	res, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Password:       "secret123",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !res.NeedsTenant {
		t.Error("with multiple memberships, NeedsTenant must be true")
	}
	if len(res.Memberships) != 2 {
		t.Errorf("expected 2 memberships, got %d", len(res.Memberships))
	}
}
