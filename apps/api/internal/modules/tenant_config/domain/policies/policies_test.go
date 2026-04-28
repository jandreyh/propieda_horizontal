package policies_test

import (
	"strings"
	"testing"

	"github.com/saas-ph/api/internal/modules/tenant_config/domain/policies"
)

func TestValidateSettingKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"snake", "contact.email", false},
		{"single_letter", "a", false},
		{"with_digits", "visits.v2_threshold", false},
		{"alphanumeric_dotted", "module1.setting9_v2", false},
		{"empty", "", true},
		{"starts_with_digit", "1invalid", true},
		{"starts_with_dot", ".invalid", true},
		{"upper_case", "Contact.Email", true},
		{"hyphen_not_allowed", "contact-email", true},
		{"space", "contact email", true},
		{"too_long", strings.Repeat("a", 129), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := policies.ValidateSettingKey(tc.key)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for key %q, got nil", tc.key)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for key %q: %v", tc.key, err)
			}
		})
	}
}

func TestValidateSettingValue(t *testing.T) {
	t.Parallel()

	if err := policies.ValidateSettingValue([]byte(`"hello"`)); err != nil {
		t.Fatalf("non-empty value rejected: %v", err)
	}
	if err := policies.ValidateSettingValue(nil); err == nil {
		t.Fatal("nil value accepted, expected error")
	}
	if err := policies.ValidateSettingValue([]byte{}); err == nil {
		t.Fatal("empty value accepted, expected error")
	}
}

func TestValidateHexColor(t *testing.T) {
	t.Parallel()

	str := func(s string) *string { return &s }

	cases := []struct {
		name    string
		in      *string
		wantErr bool
	}{
		{"nil_ok", nil, false},
		{"6digit_lower", str("#aabbcc"), false},
		{"6digit_upper", str("#AABBCC"), false},
		{"3digit", str("#abc"), false},
		{"empty_string_rejected", str(""), true},
		{"missing_hash", str("aabbcc"), true},
		{"too_long", str("#aabbcccc"), true},
		{"non_hex", str("#zzzzzz"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := policies.ValidateHexColor(tc.in)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %v, got nil", tc.in)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %v: %v", tc.in, err)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	t.Parallel()

	good := []string{"America/Bogota", "America/Cali", "America/Medellin", "America/Barranquilla", "America/Cartagena", "UTC"}
	for _, tz := range good {
		if err := policies.ValidateTimezone(tz); err != nil {
			t.Errorf("expected %q valid, got %v", tz, err)
		}
	}
	bad := []string{"", "America/New_York", "Europe/Madrid", "GMT+5", "bogota"}
	for _, tz := range bad {
		if err := policies.ValidateTimezone(tz); err == nil {
			t.Errorf("expected %q invalid, got nil", tz)
		}
	}
}

func TestAllowedTimezonesContainsColombiaAndUTC(t *testing.T) {
	t.Parallel()

	allowed := policies.AllowedTimezones()
	want := map[string]bool{
		"America/Bogota":       false,
		"America/Cali":         false,
		"America/Medellin":     false,
		"America/Barranquilla": false,
		"America/Cartagena":    false,
		"UTC":                  false,
	}
	for _, tz := range allowed {
		if _, ok := want[tz]; ok {
			want[tz] = true
		}
	}
	for tz, found := range want {
		if !found {
			t.Errorf("expected %q in allowed timezones", tz)
		}
	}
}

func TestValidateLocale(t *testing.T) {
	t.Parallel()

	good := []string{"es", "es-CO", "en-US"}
	for _, l := range good {
		if err := policies.ValidateLocale(l); err != nil {
			t.Errorf("expected %q valid, got %v", l, err)
		}
	}
	bad := []string{"", "ES-CO", "english", "es_CO", "es-co"}
	for _, l := range bad {
		if err := policies.ValidateLocale(l); err == nil {
			t.Errorf("expected %q invalid, got nil", l)
		}
	}
}

func TestValidateDisplayName(t *testing.T) {
	t.Parallel()

	if err := policies.ValidateDisplayName("Conjunto Las Acacias"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := policies.ValidateDisplayName(""); err == nil {
		t.Fatal("empty accepted")
	}
	if err := policies.ValidateDisplayName("   "); err == nil {
		t.Fatal("whitespace accepted")
	}
	if err := policies.ValidateDisplayName(strings.Repeat("x", 201)); err == nil {
		t.Fatal("too long accepted")
	}
}
