package policies_test

import (
	"testing"

	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	"github.com/saas-ph/api/internal/modules/packages/domain/policies"
)

func TestRequiresEvidence(t *testing.T) {
	t.Run("nil category does not require evidence", func(t *testing.T) {
		if policies.RequiresEvidence(nil) {
			t.Error("nil should not require evidence")
		}
	})
	t.Run("category with flag false", func(t *testing.T) {
		c := &entities.PackageCategory{Name: "Sobre", RequiresEvidence: false}
		if policies.RequiresEvidence(c) {
			t.Error("expected false")
		}
	})
	t.Run("category with flag true", func(t *testing.T) {
		c := &entities.PackageCategory{Name: "Refrigerado", RequiresEvidence: true}
		if !policies.RequiresEvidence(c) {
			t.Error("expected true")
		}
	})
}

func TestCanTransition(t *testing.T) {
	cases := []struct {
		name    string
		current entities.PackageStatus
		next    entities.PackageStatus
		want    bool
	}{
		{"received -> delivered", entities.PackageStatusReceived, entities.PackageStatusDelivered, true},
		{"received -> returned", entities.PackageStatusReceived, entities.PackageStatusReturned, true},
		{"received -> received NOT allowed", entities.PackageStatusReceived, entities.PackageStatusReceived, false},
		{"delivered -> received NOT allowed", entities.PackageStatusDelivered, entities.PackageStatusReceived, false},
		{"delivered -> returned NOT allowed", entities.PackageStatusDelivered, entities.PackageStatusReturned, false},
		{"returned -> delivered NOT allowed", entities.PackageStatusReturned, entities.PackageStatusDelivered, false},
		{"returned -> received NOT allowed", entities.PackageStatusReturned, entities.PackageStatusReceived, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policies.CanTransition(tc.current, tc.next)
			if got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	if err := policies.ValidateUUID("11111111-2222-3333-4444-555555555555"); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if err := policies.ValidateUUID("bad"); err == nil {
		t.Error("expected error")
	}
	if err := policies.ValidateUUID(""); err == nil {
		t.Error("expected error")
	}
}

func TestValidateRecipientName(t *testing.T) {
	if err := policies.ValidateRecipientName(""); err == nil {
		t.Error("empty should be invalid")
	}
	if err := policies.ValidateRecipientName("Juan Perez"); err != nil {
		t.Errorf("got %v", err)
	}
	long := make([]byte, 250)
	for i := range long {
		long[i] = 'x'
	}
	if err := policies.ValidateRecipientName(string(long)); err == nil {
		t.Error(">200 should be invalid")
	}
}
