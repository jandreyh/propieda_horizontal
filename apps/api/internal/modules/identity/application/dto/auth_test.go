package dto_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
)

func TestLoginRequestJSON(t *testing.T) {
	raw := `{"identifier":"CC:12345","password":"s3cret"}`
	var req dto.LoginRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Identifier != "CC:12345" || req.Password != "s3cret" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestLoginResponseJSONOmitEmpty(t *testing.T) {
	resp := dto.LoginResponse{MFARequired: true, PreAuthToken: "tok"}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"mfa_required":true`) {
		t.Fatalf("missing mfa_required: %s", s)
	}
	if !strings.Contains(s, `"pre_auth_token":"tok"`) {
		t.Fatalf("missing pre_auth_token: %s", s)
	}
	if strings.Contains(s, "access_token") || strings.Contains(s, "refresh_token") {
		t.Fatalf("should omit empty token fields: %s", s)
	}
}

func TestLoginResponseJSONFullSession(t *testing.T) {
	resp := dto.LoginResponse{
		AccessToken:  "a",
		RefreshToken: "r",
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{`"access_token":"a"`, `"refresh_token":"r"`, `"expires_in":900`, `"token_type":"Bearer"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
}

func TestMeResponseJSON(t *testing.T) {
	email := "u@example.com"
	at := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	r := dto.MeResponse{
		ID:             "id-1",
		DocumentType:   "CC",
		DocumentNumber: "123",
		Names:          "Ana",
		LastNames:      "Perez",
		Email:          &email,
		MFAEnrolledAt:  &at,
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"email":"u@example.com"`) {
		t.Fatalf("missing email: %s", s)
	}
	if strings.Contains(s, "phone") {
		t.Fatalf("phone should be omitted: %s", s)
	}
}

func TestMFAVerifyRequestRoundtrip(t *testing.T) {
	in := dto.MFAVerifyRequest{PreAuthToken: "pat", Code: "123456"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out dto.MFAVerifyRequest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", in, out)
	}
}
