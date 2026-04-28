package policies_test

import (
	"testing"
	"time"

	"github.com/saas-ph/api/internal/modules/units/domain/entities"
	"github.com/saas-ph/api/internal/modules/units/domain/policies"
)

func ptrTime(t time.Time) *time.Time { return &t }

func activeOwner(pct float64) entities.UnitOwner {
	return entities.UnitOwner{Percentage: pct}
}

func terminatedOwner(pct float64) entities.UnitOwner {
	until := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	return entities.UnitOwner{Percentage: pct, UntilDate: ptrTime(until)}
}

func TestValidatePercentageRange(t *testing.T) {
	cases := []struct {
		in   float64
		want bool
	}{
		{0, false},
		{-1, false},
		{0.0001, true},
		{50, true},
		{100, true},
		{100.01, false},
	}
	for _, tc := range cases {
		if got := policies.ValidatePercentageRange(tc.in); got != tc.want {
			t.Fatalf("ValidatePercentageRange(%v) = %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestValidatePercentage_GoldenSumWithinLimit(t *testing.T) {
	current := []entities.UnitOwner{activeOwner(60)}
	if !policies.ValidatePercentage(current, 40) {
		t.Fatalf("60+40=100 should be valid")
	}
}

func TestValidatePercentage_ExceedsLimit(t *testing.T) {
	current := []entities.UnitOwner{activeOwner(60)}
	if policies.ValidatePercentage(current, 41) {
		t.Fatalf("60+41=101 should be rejected")
	}
}

func TestValidatePercentage_IgnoresTerminatedOwners(t *testing.T) {
	current := []entities.UnitOwner{
		activeOwner(50),
		terminatedOwner(50), // ya no cuenta
	}
	if !policies.ValidatePercentage(current, 50) {
		t.Fatalf("terminated owners must not count toward the limit")
	}
}

func TestValidatePercentage_NewPctOutOfRange(t *testing.T) {
	current := []entities.UnitOwner{}
	if policies.ValidatePercentage(current, 0) {
		t.Fatalf("0%% must be rejected")
	}
	if policies.ValidatePercentage(current, 100.5) {
		t.Fatalf(">100%% must be rejected even with empty list")
	}
}

func TestSumActivePercentages_FiltersTerminated(t *testing.T) {
	got := policies.SumActivePercentages([]entities.UnitOwner{
		activeOwner(30),
		activeOwner(20),
		terminatedOwner(99),
	})
	want := 50.0
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestEnsureOnlyOnePrimary_NoPrimary(t *testing.T) {
	active := []entities.UnitOccupancy{
		{IsPrimary: false},
		{IsPrimary: false},
	}
	if !policies.EnsureOnlyOnePrimary(active, false) {
		t.Fatalf("zero primaries is fine")
	}
}

func TestEnsureOnlyOnePrimary_OneExisting(t *testing.T) {
	active := []entities.UnitOccupancy{
		{IsPrimary: true},
		{IsPrimary: false},
	}
	if !policies.EnsureOnlyOnePrimary(active, false) {
		t.Fatalf("one primary should be valid")
	}
}

func TestEnsureOnlyOnePrimary_AddingSecondPrimary(t *testing.T) {
	active := []entities.UnitOccupancy{
		{IsPrimary: true},
	}
	if policies.EnsureOnlyOnePrimary(active, true) {
		t.Fatalf("adding a second primary must violate the invariant")
	}
}

func TestEnsureOnlyOnePrimary_AddingFirstPrimary(t *testing.T) {
	active := []entities.UnitOccupancy{
		{IsPrimary: false},
		{IsPrimary: false},
	}
	if !policies.EnsureOnlyOnePrimary(active, true) {
		t.Fatalf("adding first primary should be valid")
	}
}

func TestEnsureOnlyOnePrimary_IgnoresInactive(t *testing.T) {
	moved := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	active := []entities.UnitOccupancy{
		{IsPrimary: true, MoveOutDate: &moved}, // ya no cuenta
		{IsPrimary: false},
	}
	if !policies.EnsureOnlyOnePrimary(active, true) {
		t.Fatalf("a moved-out primary must not block a new primary")
	}
}
