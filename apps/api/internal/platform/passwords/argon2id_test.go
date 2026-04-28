package passwords

import (
	"errors"
	"strings"
	"testing"
)

func TestHashAndVerifyRoundtrip(t *testing.T) {
	t.Parallel()
	encoded, err := Hash("super-secret-pw")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$") {
		t.Fatalf("expected argon2id prefix, got %q", encoded)
	}
	if err := Verify("super-secret-pw", encoded); err != nil {
		t.Fatalf("Verify ok: %v", err)
	}
	if err := Verify("wrong", encoded); !errors.Is(err, ErrMismatch) {
		t.Fatalf("Verify wrong: expected ErrMismatch, got %v", err)
	}
}

func TestVerifyRejectsUnsupportedAlgo(t *testing.T) {
	t.Parallel()
	bcryptish := "$2y$10$abcdefghijabcdefghijab"
	if err := Verify("x", bcryptish); !errors.Is(err, ErrUnsupportedAlgo) {
		t.Fatalf("expected ErrUnsupportedAlgo, got %v", err)
	}
}

func TestHashRejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, err := Hash(""); err == nil {
		t.Fatalf("expected error for empty plain")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	t.Parallel()
	cases := []string{
		"$argon2id$",
		"$argon2id$v=19$m=64,t=3,p=2$invalid$invalid",
	}
	for _, c := range cases {
		if err := Verify("x", c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}
