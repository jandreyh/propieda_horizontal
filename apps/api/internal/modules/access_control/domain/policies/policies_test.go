package policies_test

import (
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	"github.com/saas-ph/api/internal/modules/access_control/domain/policies"
)

func TestIsExpired(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		t    time.Time
		now  time.Time
		want bool
	}{
		{"zero is not expired", time.Time{}, now, false},
		{"future is not expired", now.Add(time.Hour), now, false},
		{"past is expired", now.Add(-time.Hour), now, true},
		{"equal is expired", now, now, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policies.IsExpired(tc.t, tc.now)
			if got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestRequiresPhoto(t *testing.T) {
	if !policies.RequiresPhoto("manual") {
		t.Error("manual should require photo")
	}
	if policies.RequiresPhoto("qr") {
		t.Error("qr should NOT require photo")
	}
	if policies.RequiresPhoto("") {
		t.Error("empty source should NOT require photo")
	}
}

func TestBlacklistMatch(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	t.Run("active without expiry matches", func(t *testing.T) {
		b := entities.BlacklistEntry{Status: entities.BlacklistStatusActive}
		if !policies.BlacklistMatch(b, now) {
			t.Error("expected match")
		}
	})

	t.Run("active with future expiry matches", func(t *testing.T) {
		b := entities.BlacklistEntry{
			Status:    entities.BlacklistStatusActive,
			ExpiresAt: &future,
		}
		if !policies.BlacklistMatch(b, now) {
			t.Error("expected match")
		}
	})

	t.Run("active with past expiry does NOT match", func(t *testing.T) {
		b := entities.BlacklistEntry{
			Status:    entities.BlacklistStatusActive,
			ExpiresAt: &past,
		}
		if policies.BlacklistMatch(b, now) {
			t.Error("expected no match (expired)")
		}
	})

	t.Run("archived does NOT match", func(t *testing.T) {
		b := entities.BlacklistEntry{Status: entities.BlacklistStatusArchived}
		if policies.BlacklistMatch(b, now) {
			t.Error("expected no match (archived)")
		}
	})

	t.Run("soft-deleted does NOT match", func(t *testing.T) {
		t0 := past
		b := entities.BlacklistEntry{
			Status:    entities.BlacklistStatusActive,
			DeletedAt: &t0,
		}
		if policies.BlacklistMatch(b, now) {
			t.Error("expected no match (soft-deleted)")
		}
	})
}

func TestValidateDocumentType(t *testing.T) {
	for _, ok := range []string{"CC", "CE", "PA", "TI", "RC", "NIT"} {
		if err := policies.ValidateDocumentType(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "XX", "cc"} {
		if err := policies.ValidateDocumentType(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestValidateMaxUses(t *testing.T) {
	v, err := policies.ValidateMaxUses(nil)
	if err != nil || v != 1 {
		t.Errorf("nil should default to 1, got %d err=%v", v, err)
	}
	good := int32(5)
	v, err = policies.ValidateMaxUses(&good)
	if err != nil || v != 5 {
		t.Errorf("got %d err=%v", v, err)
	}
	bad := int32(0)
	if _, err := policies.ValidateMaxUses(&bad); err == nil {
		t.Error("0 should be invalid")
	}
}
