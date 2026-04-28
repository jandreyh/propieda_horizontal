package policies_test

import (
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/announcements/domain/entities"
	"github.com/saas-ph/api/internal/modules/announcements/domain/policies"
)

// --- IsExpired ---

func TestIsExpired(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"nil_does_not_expire", nil, false},
		{"future_not_expired", &future, false},
		{"past_expired", &past, true},
		{"exactly_now_expired", &now, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policies.IsExpired(now, tc.expiresAt)
			if got != tc.want {
				t.Fatalf("IsExpired(%v, %v) = %v, want %v", now, tc.expiresAt, got, tc.want)
			}
		})
	}
}

// --- MatchesAudience ---

func ptr(s string) *string { return &s }

func TestMatchesAudience(t *testing.T) {
	scopes := policies.UserScopes{
		RoleIDs:      []string{"role-1", "role-2"},
		StructureIDs: []string{"struct-A"},
		UnitIDs:      []string{"unit-X"},
	}

	cases := []struct {
		name      string
		audiences []entities.Audience
		want      bool
	}{
		{
			name: "global_match",
			audiences: []entities.Audience{
				{TargetType: entities.TargetGlobal, TargetID: nil},
			},
			want: true,
		},
		{
			name: "role_match",
			audiences: []entities.Audience{
				{TargetType: entities.TargetRole, TargetID: ptr("role-2")},
			},
			want: true,
		},
		{
			name: "structure_match",
			audiences: []entities.Audience{
				{TargetType: entities.TargetStructure, TargetID: ptr("struct-A")},
			},
			want: true,
		},
		{
			name: "unit_match",
			audiences: []entities.Audience{
				{TargetType: entities.TargetUnit, TargetID: ptr("unit-X")},
			},
			want: true,
		},
		{
			name: "no_match",
			audiences: []entities.Audience{
				{TargetType: entities.TargetRole, TargetID: ptr("role-other")},
				{TargetType: entities.TargetUnit, TargetID: ptr("unit-other")},
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policies.MatchesAudience(tc.audiences, scopes)
			if got != tc.want {
				t.Fatalf("MatchesAudience() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ValidateAudienceCoherence ---

func TestValidateAudienceCoherence(t *testing.T) {
	t.Run("global_with_nil_id_ok", func(t *testing.T) {
		if err := policies.ValidateAudienceCoherence(entities.TargetGlobal, nil); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("global_with_id_fails", func(t *testing.T) {
		if err := policies.ValidateAudienceCoherence(entities.TargetGlobal, ptr("xxx")); err == nil {
			t.Error("expected error for global with target_id set")
		}
	})
	t.Run("role_without_id_fails", func(t *testing.T) {
		if err := policies.ValidateAudienceCoherence(entities.TargetRole, nil); err == nil {
			t.Error("expected error for role without target_id")
		}
	})
	t.Run("role_with_id_ok", func(t *testing.T) {
		if err := policies.ValidateAudienceCoherence(entities.TargetRole, ptr("role-1")); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("invalid_target_type", func(t *testing.T) {
		if err := policies.ValidateAudienceCoherence(entities.TargetType("garbage"), ptr("x")); err == nil {
			t.Error("expected error for invalid target_type")
		}
	})
}
