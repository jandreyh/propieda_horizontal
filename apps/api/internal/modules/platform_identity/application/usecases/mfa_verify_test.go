package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
	"github.com/saas-ph/api/internal/platform/jwtsign"
)

func freshTOTPSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "ph-saas", AccountName: "test@example.com"})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}
	return key.Secret()
}

func TestMFAVerify_Success(t *testing.T) {
	secret := freshTOTPSecret(t)
	user := newActiveUser(t, "ana@demo.test", "x", true)
	user.MFASecret = &secret
	users := &fakeRepoForRefresh{
		fakeRepo: &fakeRepo{
			memberships: []entities.Membership{{TenantID: uuid.New(), TenantSlug: "demo", TenantName: "Demo", Role: "tenant_admin", Status: "active"}},
		},
		users: map[uuid.UUID]*entities.PlatformUser{user.ID: user},
	}
	signer := newTestSigner(t)
	sessions := newFakeSessions()
	uc := NewMFAVerifyUseCase(MFAVerifyDeps{Users: users, Sessions: sessions, Signer: signer})

	preAuth, err := signer.Sign(user.ID.String(), "", PreAuthSessionMarker, []string{PreAuthRole}, []string{"pwd"}, time.Minute)
	if err != nil {
		t.Fatalf("sign pre-auth: %v", err)
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("gen code: %v", err)
	}

	res, err := uc.Execute(context.Background(), dto.MFAVerifyRequest{PreAuthToken: preAuth, Code: code})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Error("expected access + refresh tokens")
	}
}

func TestMFAVerify_InvalidPreAuth(t *testing.T) {
	users := &fakeRepoForRefresh{fakeRepo: &fakeRepo{}, users: map[uuid.UUID]*entities.PlatformUser{}}
	uc := NewMFAVerifyUseCase(MFAVerifyDeps{Users: users, Sessions: newFakeSessions(), Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.MFAVerifyRequest{PreAuthToken: "not-a-jwt", Code: "123456"})
	if !errors.Is(err, ErrInvalidPreAuth) {
		t.Fatalf("expected ErrInvalidPreAuth, got %v", err)
	}
}

func TestMFAVerify_NormalAccessTokenRejected(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "x", true)
	users := &fakeRepoForRefresh{fakeRepo: &fakeRepo{}, users: map[uuid.UUID]*entities.PlatformUser{user.ID: user}}
	signer := newTestSigner(t)
	uc := NewMFAVerifyUseCase(MFAVerifyDeps{Users: users, Sessions: newFakeSessions(), Signer: signer})

	// Token con sid normal (no PreAuthSessionMarker) — debe rechazarse.
	normal, _ := signer.SignPlatform(user.ID.String(), "real-session", "", []jwtsign.MembershipClaim{}, []string{"pwd"}, time.Minute)
	_, err := uc.Execute(context.Background(), dto.MFAVerifyRequest{PreAuthToken: normal, Code: "000000"})
	if !errors.Is(err, ErrInvalidPreAuth) {
		t.Fatalf("expected ErrInvalidPreAuth, got %v", err)
	}
}

func TestMFAVerify_BadCode(t *testing.T) {
	secret := freshTOTPSecret(t)
	user := newActiveUser(t, "ana@demo.test", "x", true)
	user.MFASecret = &secret
	users := &fakeRepoForRefresh{fakeRepo: &fakeRepo{}, users: map[uuid.UUID]*entities.PlatformUser{user.ID: user}}
	signer := newTestSigner(t)
	uc := NewMFAVerifyUseCase(MFAVerifyDeps{Users: users, Sessions: newFakeSessions(), Signer: signer})

	preAuth, _ := signer.Sign(user.ID.String(), "", PreAuthSessionMarker, []string{PreAuthRole}, []string{"pwd"}, time.Minute)
	_, err := uc.Execute(context.Background(), dto.MFAVerifyRequest{PreAuthToken: preAuth, Code: "000000"})
	if !errors.Is(err, ErrInvalidMFACode) {
		t.Fatalf("expected ErrInvalidMFACode, got %v", err)
	}
}

func TestMFAVerify_EmptyInput(t *testing.T) {
	users := &fakeRepoForRefresh{fakeRepo: &fakeRepo{}, users: map[uuid.UUID]*entities.PlatformUser{}}
	uc := NewMFAVerifyUseCase(MFAVerifyDeps{Users: users, Sessions: newFakeSessions(), Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.MFAVerifyRequest{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
