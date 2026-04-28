package policies_test

import (
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
	"github.com/saas-ph/api/internal/modules/identity/domain/policies"
)

func ptrTime(t time.Time) *time.Time { return &t }

func ptrString(s string) *string { return &s }

func TestIsLocked(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		user *entities.User
		want bool
	}{
		{
			name: "nil user is not locked",
			user: nil,
			want: false,
		},
		{
			name: "no locked_until is not locked",
			user: &entities.User{},
			want: false,
		},
		{
			name: "locked_until in the future is locked",
			user: &entities.User{LockedUntil: ptrTime(now.Add(5 * time.Minute))},
			want: true,
		},
		{
			name: "locked_until in the past is not locked",
			user: &entities.User{LockedUntil: ptrTime(now.Add(-1 * time.Minute))},
			want: false,
		},
		{
			name: "locked_until equal to now is not locked",
			user: &entities.User{LockedUntil: ptrTime(now)},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := policies.IsLocked(tc.user, now)
			if got != tc.want {
				t.Fatalf("IsLocked = %v want %v", got, tc.want)
			}
		})
	}
}

func TestCanLogin(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	active := func() *entities.User {
		return &entities.User{Status: entities.UserStatusActive}
	}

	t.Run("nil user", func(t *testing.T) {
		if policies.CanLogin(nil, now) {
			t.Fatal("nil user should not be able to login")
		}
	})
	t.Run("active user without lockout", func(t *testing.T) {
		if !policies.CanLogin(active(), now) {
			t.Fatal("active user without lockout should be able to login")
		}
	})
	t.Run("inactive user", func(t *testing.T) {
		u := active()
		u.Status = entities.UserStatusInactive
		if policies.CanLogin(u, now) {
			t.Fatal("inactive user should not be able to login")
		}
	})
	t.Run("suspended user", func(t *testing.T) {
		u := active()
		u.Status = entities.UserStatusSuspended
		if policies.CanLogin(u, now) {
			t.Fatal("suspended user should not be able to login")
		}
	})
	t.Run("soft-deleted user", func(t *testing.T) {
		u := active()
		u.DeletedAt = ptrTime(now.Add(-time.Hour))
		if policies.CanLogin(u, now) {
			t.Fatal("soft-deleted user should not be able to login")
		}
	})
	t.Run("user with active lockout", func(t *testing.T) {
		u := active()
		u.LockedUntil = ptrTime(now.Add(5 * time.Minute))
		if policies.CanLogin(u, now) {
			t.Fatal("locked user should not be able to login")
		}
	})
}

func TestWouldExceedFailedAttempts(t *testing.T) {
	cases := []struct {
		current int
		want    bool
	}{
		{0, false},
		{1, false},
		{2, false},
		{3, false},
		{4, true}, // current=4 + 1 = 5, hits the threshold
		{5, true},
		{99, true},
	}
	for _, tc := range cases {
		if got := policies.WouldExceedFailedAttempts(tc.current); got != tc.want {
			t.Fatalf("WouldExceedFailedAttempts(%d) = %v want %v", tc.current, got, tc.want)
		}
	}
}

func TestNextLockUntil(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	got := policies.NextLockUntil(now)
	want := now.Add(policies.LockoutDuration)
	if !got.Equal(want) {
		t.Fatalf("NextLockUntil = %v want %v", got, want)
	}
}

func TestShouldRequireMFA(t *testing.T) {
	t.Run("nil user", func(t *testing.T) {
		if policies.ShouldRequireMFA(nil) {
			t.Fatal("nil user does not require MFA")
		}
	})
	t.Run("user without secret", func(t *testing.T) {
		if policies.ShouldRequireMFA(&entities.User{}) {
			t.Fatal("user without secret does not require MFA")
		}
	})
	t.Run("user with empty secret pointer", func(t *testing.T) {
		empty := ""
		if policies.ShouldRequireMFA(&entities.User{MFASecret: &empty}) {
			t.Fatal("user with empty secret does not require MFA")
		}
	})
	t.Run("user with MFA secret", func(t *testing.T) {
		if !policies.ShouldRequireMFA(&entities.User{MFASecret: ptrString("JBSWY3DPEHPK3PXP")}) {
			t.Fatal("user with secret should require MFA")
		}
	})
}
