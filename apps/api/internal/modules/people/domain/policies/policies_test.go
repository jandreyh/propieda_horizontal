package policies_test

import (
	"testing"

	"github.com/saas-ph/api/internal/modules/people/domain/policies"
)

func TestNormalizePlate(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"abc123", "ABC123"},
		{"  abc123  ", "ABC123"},
		{"\tABC12a\n", "ABC12A"},
		{"AbC12A", "ABC12A"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := policies.NormalizePlate(tc.in)
			if got != tc.want {
				t.Fatalf("NormalizePlate(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsValidColombianPlate(t *testing.T) {
	t.Run("valid_car_plate", func(t *testing.T) {
		valid := []string{"ABC123", "ZZZ999", "AAA000"}
		for _, p := range valid {
			if !policies.IsValidColombianPlate(p) {
				t.Errorf("expected %q to be valid car plate", p)
			}
		}
	})

	t.Run("valid_motorcycle_plate", func(t *testing.T) {
		valid := []string{"ABC12A", "ZZZ99Z", "AAA00B"}
		for _, p := range valid {
			if !policies.IsValidColombianPlate(p) {
				t.Errorf("expected %q to be valid motorcycle plate", p)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		invalid := []string{
			"",        // empty
			"abc123",  // lowercase (should already be normalized)
			"AB123",   // too short
			"ABCD123", // too many letters
			"ABC1234", // too many digits
			"ABC1A2",  // wrong pattern
			"123ABC",  // digits first
			"ABC-123", // separator
			"ABC 123", // space
			"ABC12AB", // 3 letters + 2 digits + 2 letters
		}
		for _, p := range invalid {
			if policies.IsValidColombianPlate(p) {
				t.Errorf("expected %q to be invalid", p)
			}
		}
	})
}

func TestValidatePlate(t *testing.T) {
	t.Run("normalizes_and_validates", func(t *testing.T) {
		got, err := policies.ValidatePlate("  abc123  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "ABC123" {
			t.Fatalf("want ABC123, got %q", got)
		}
	})

	t.Run("rejects_empty", func(t *testing.T) {
		if _, err := policies.ValidatePlate("   "); err == nil {
			t.Fatal("expected error for empty plate")
		}
	})

	t.Run("rejects_invalid_format", func(t *testing.T) {
		if _, err := policies.ValidatePlate("XX-1"); err == nil {
			t.Fatal("expected error for invalid format")
		}
	})
}

func TestValidateVehicleType(t *testing.T) {
	allowed := []string{"car", "motorcycle", "truck", "bicycle", "other"}
	for _, t1 := range allowed {
		if err := policies.ValidateVehicleType(t1); err != nil {
			t.Errorf("expected %q to be allowed, got %v", t1, err)
		}
	}
	if err := policies.ValidateVehicleType("plane"); err == nil {
		t.Error("expected error for unknown type")
	}
	if err := policies.ValidateVehicleType(""); err == nil {
		t.Error("expected error for empty type")
	}
}

func TestValidateVehicleYear(t *testing.T) {
	if err := policies.ValidateVehicleYear(nil); err != nil {
		t.Errorf("nil should be allowed, got %v", err)
	}
	ok := int32(2024)
	if err := policies.ValidateVehicleYear(&ok); err != nil {
		t.Errorf("2024 should be allowed, got %v", err)
	}
	tooLow := int32(1900)
	if err := policies.ValidateVehicleYear(&tooLow); err == nil {
		t.Error("expected error for year=1900")
	}
	tooHigh := int32(2200)
	if err := policies.ValidateVehicleYear(&tooHigh); err == nil {
		t.Error("expected error for year=2200")
	}
}

func TestValidateUUID(t *testing.T) {
	ok := "11111111-2222-3333-4444-555555555555"
	if err := policies.ValidateUUID(ok); err != nil {
		t.Errorf("expected %q to be valid uuid: %v", ok, err)
	}
	bad := []string{
		"",
		"not-a-uuid",
		"11111111-2222-3333-4444-55555555555",   // too short
		"11111111-2222-3333-4444-5555555555555", // too long
		"11111111X2222-3333-4444-555555555555",  // missing dash
		"zzzzzzzz-2222-3333-4444-555555555555",  // non-hex
	}
	for _, s := range bad {
		if err := policies.ValidateUUID(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}
