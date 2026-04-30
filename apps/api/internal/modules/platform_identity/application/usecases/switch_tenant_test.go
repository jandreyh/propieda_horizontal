package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
)

func TestSwitchTenant_Success(t *testing.T) {
	tenantID := uuid.New()
	repo := &fakeRepo{
		memberships: []entities.Membership{
			{TenantID: tenantID, TenantSlug: "demo", TenantName: "Conjunto Demo", Role: "tenant_admin", Status: "active"},
			{TenantID: uuid.New(), TenantSlug: "demo2", TenantName: "Demo 2", Role: "guard", Status: "active"},
		},
	}
	uc := NewSwitchTenantUseCase(SwitchTenantDeps{Users: repo, Signer: newTestSigner(t)})

	res, err := uc.Execute(context.Background(), uuid.New().String(), dto.SwitchTenantRequest{TenantSlug: "DEMO"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.AccessToken == "" {
		t.Error("expected access_token")
	}
	if res.CurrentTenant.TenantSlug != "demo" {
		t.Errorf("CurrentTenant.slug = %q", res.CurrentTenant.TenantSlug)
	}
	if res.CurrentTenant.Role != "tenant_admin" {
		t.Errorf("CurrentTenant.role = %q", res.CurrentTenant.Role)
	}
}

func TestSwitchTenant_NoMembership(t *testing.T) {
	repo := &fakeRepo{
		memberships: []entities.Membership{
			{TenantID: uuid.New(), TenantSlug: "demo", TenantName: "Demo", Role: "tenant_admin", Status: "active"},
		},
	}
	uc := NewSwitchTenantUseCase(SwitchTenantDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), uuid.New().String(), dto.SwitchTenantRequest{TenantSlug: "ghost"})
	if !errors.Is(err, ErrMembershipMissing) {
		t.Fatalf("expected ErrMembershipMissing, got %v", err)
	}
}

func TestSwitchTenant_InvalidInput(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewSwitchTenantUseCase(SwitchTenantDeps{Users: repo, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), uuid.New().String(), dto.SwitchTenantRequest{TenantSlug: "  "})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	_, err = uc.Execute(context.Background(), "not-a-uuid", dto.SwitchTenantRequest{TenantSlug: "demo"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
