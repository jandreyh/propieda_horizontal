package usecases_test

import (
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/application/usecases"
)

func freshTOTPSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "ph-saas", AccountName: "test@example.com"})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}
	return key.Secret()
}

func TestMFAVerifyUseCase_GoldenPath(t *testing.T) {
	signer := newSigner(t)
	users := newUserRepoMock()
	sessions := newSessionRepoMock()
	secret := freshTOTPSecret(t)
	user := newActiveUser(t, "S3cret!")
	user.MFASecret = &secret
	users.put(user)

	// emitir pre_auth token con mismo signer
	pat, err := signer.Sign(user.ID, "tenant-123", usecases.PreAuthSessionMarker, []string{usecases.PreAuthRole}, []string{"pwd"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("Sign pat: %v", err)
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	uc := usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
		Users:    users,
		Sessions: sessions,
		Signer:   signer,
	})
	resp, err := uc.Execute(newTenantCtx(), dto.MFAVerifyRequest{PreAuthToken: pat, Code: code})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", resp)
	}
}

func TestMFAVerifyUseCase_InvalidInput(t *testing.T) {
	uc := usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
	})
	_, err := uc.Execute(newTenantCtx(), dto.MFAVerifyRequest{})
	if !errors.Is(err, usecases.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestMFAVerifyUseCase_BadPreAuthToken(t *testing.T) {
	uc := usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   newSigner(t),
	})
	_, err := uc.Execute(newTenantCtx(), dto.MFAVerifyRequest{PreAuthToken: "garbage", Code: "123456"})
	if !errors.Is(err, usecases.ErrInvalidPreAuth) {
		t.Fatalf("expected ErrInvalidPreAuth, got %v", err)
	}
}

func TestMFAVerifyUseCase_TokenWithoutPreAuthMarker(t *testing.T) {
	signer := newSigner(t)
	// firmar un token de sesion normal — no pre-auth
	tok, _ := signer.Sign("user-1", "tenant-123", "real-session", nil, []string{"pwd"}, 5*time.Minute)
	uc := usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   signer,
	})
	_, err := uc.Execute(newTenantCtx(), dto.MFAVerifyRequest{PreAuthToken: tok, Code: "000000"})
	if !errors.Is(err, usecases.ErrInvalidPreAuth) {
		t.Fatalf("expected ErrInvalidPreAuth, got %v", err)
	}
}

func TestMFAVerifyUseCase_BadCode(t *testing.T) {
	signer := newSigner(t)
	users := newUserRepoMock()
	secret := freshTOTPSecret(t)
	user := newActiveUser(t, "S3cret!")
	user.MFASecret = &secret
	users.put(user)
	pat, _ := signer.Sign(user.ID, "tenant-123", usecases.PreAuthSessionMarker, []string{usecases.PreAuthRole}, nil, 5*time.Minute)
	uc := usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
		Users:    users,
		Sessions: newSessionRepoMock(),
		Signer:   signer,
	})
	_, err := uc.Execute(newTenantCtx(), dto.MFAVerifyRequest{PreAuthToken: pat, Code: "000000"})
	if !errors.Is(err, usecases.ErrInvalidMFACode) {
		t.Fatalf("expected ErrInvalidMFACode, got %v", err)
	}
}

func TestMFAVerifyUseCase_UserNotFound(t *testing.T) {
	signer := newSigner(t)
	pat, _ := signer.Sign("ghost-user", "tenant-123", usecases.PreAuthSessionMarker, []string{usecases.PreAuthRole}, nil, 5*time.Minute)
	uc := usecases.NewMFAVerifyUseCase(usecases.MFAVerifyDeps{
		Users:    newUserRepoMock(),
		Sessions: newSessionRepoMock(),
		Signer:   signer,
	})
	code, _ := totp.GenerateCode("JBSWY3DPEHPK3PXP", time.Now())
	_, err := uc.Execute(newTenantCtx(), dto.MFAVerifyRequest{PreAuthToken: pat, Code: code})
	if !errors.Is(err, usecases.ErrInvalidPreAuth) {
		t.Fatalf("expected ErrInvalidPreAuth, got %v", err)
	}
}
