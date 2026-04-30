package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
)

// extender fakeRepo de login_test para soportar lookup por id.
type fakeRepoByID struct {
	*fakeRepo
	byID map[uuid.UUID]*entities.PlatformUser
}

func (f *fakeRepoByID) FindByID(_ context.Context, id uuid.UUID) (*entities.PlatformUser, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func TestMe_Success(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	repo := &fakeRepoByID{
		fakeRepo: &fakeRepo{},
		byID:     map[uuid.UUID]*entities.PlatformUser{user.ID: user},
	}
	uc := NewMeUseCase(MeDeps{Users: repo})

	res, err := uc.Execute(context.Background(), user.ID.String())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.ID != user.ID.String() || res.Email != user.Email {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMe_InvalidSubject(t *testing.T) {
	repo := &fakeRepoByID{fakeRepo: &fakeRepo{}, byID: map[uuid.UUID]*entities.PlatformUser{}}
	uc := NewMeUseCase(MeDeps{Users: repo})

	_, err := uc.Execute(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestMe_UserNotFound(t *testing.T) {
	repo := &fakeRepoByID{fakeRepo: &fakeRepo{}, byID: map[uuid.UUID]*entities.PlatformUser{}}
	uc := NewMeUseCase(MeDeps{Users: repo})

	_, err := uc.Execute(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrUserMismatch) {
		t.Fatalf("expected ErrUserMismatch, got %v", err)
	}
}

func TestMe_AccountInactive(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "x", false)
	user.Status = "suspended"
	repo := &fakeRepoByID{
		fakeRepo: &fakeRepo{},
		byID:     map[uuid.UUID]*entities.PlatformUser{user.ID: user},
	}
	uc := NewMeUseCase(MeDeps{Users: repo})

	_, err := uc.Execute(context.Background(), user.ID.String())
	if !errors.Is(err, ErrAccountInactive) {
		t.Fatalf("expected ErrAccountInactive, got %v", err)
	}
}

func TestListMemberships_Success(t *testing.T) {
	repo := &fakeRepo{
		memberships: []entities.Membership{
			{TenantID: uuid.New(), TenantSlug: "demo", TenantName: "Demo", Role: "tenant_admin", Status: "active"},
			{TenantID: uuid.New(), TenantSlug: "demo2", TenantName: "Demo 2", Role: "guard", Status: "active"},
		},
	}
	uc := NewListMembershipsUseCase(ListMembershipsDeps{Users: repo})

	res, err := uc.Execute(context.Background(), uuid.New().String())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(res.Items))
	}
}

func TestListMemberships_InvalidSubject(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewListMembershipsUseCase(ListMembershipsDeps{Users: repo})

	_, err := uc.Execute(context.Background(), "bogus")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
