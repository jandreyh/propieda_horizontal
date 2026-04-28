package jwtsign

import (
	"strings"
	"testing"
	"time"
)

func TestSignAndVerifyRoundtrip(t *testing.T) {
	t.Parallel()
	signer, err := NewSigner(SignerConfig{Issuer: "test", Audience: "test-aud"})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	tok, err := signer.Sign("user-1", "tenant-1", "sess-1", []string{"tenant_admin"}, []string{"pwd", "totp"}, time.Minute)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !strings.Contains(tok, ".") {
		t.Fatalf("token shape: %s", tok)
	}
	claims, err := signer.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "user-1" || claims.TenantID != "tenant-1" || claims.SessionID != "sess-1" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	t.Parallel()
	signer, _ := NewSigner(SignerConfig{ClockSkew: time.Nanosecond})
	tok, err := signer.Sign("u", "t", "s", nil, nil, time.Millisecond)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := signer.Verify(tok); err == nil {
		t.Fatalf("expected expired error")
	}
}

func TestVerifyRejectsTampered(t *testing.T) {
	t.Parallel()
	signer, _ := NewSigner(SignerConfig{})
	tok, _ := signer.Sign("u", "t", "s", nil, nil, time.Minute)
	if _, err := signer.Verify(tok + "x"); err == nil {
		t.Fatalf("expected tampered error")
	}
}
