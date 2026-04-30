package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/tenant_members/application/dto"
	"github.com/saas-ph/api/internal/modules/tenant_members/domain"
	"github.com/saas-ph/api/internal/modules/tenant_members/domain/entities"
)

type fakeLinks struct {
	created []domain.CreateLink
	links   map[uuid.UUID]*entities.TenantMember
	createErr error
}

func newFakeLinks() *fakeLinks {
	return &fakeLinks{links: map[uuid.UUID]*entities.TenantMember{}}
}

func (f *fakeLinks) Create(_ context.Context, in domain.CreateLink) (*entities.TenantMember, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = append(f.created, in)
	m := &entities.TenantMember{
		ID:             uuid.New(),
		PlatformUserID: in.PlatformUserID,
		Role:           in.Role,
		PrimaryUnitID:  in.PrimaryUnitID,
		Status:         "active",
		Version:        1,
	}
	f.links[m.ID] = m
	return m, nil
}

func (f *fakeLinks) List(context.Context) ([]entities.TenantMember, error) {
	out := make([]entities.TenantMember, 0, len(f.links))
	for _, m := range f.links {
		out = append(out, *m)
	}
	return out, nil
}

func (f *fakeLinks) FindByID(_ context.Context, id uuid.UUID) (*entities.TenantMember, error) {
	if m, ok := f.links[id]; ok {
		return m, nil
	}
	return nil, domain.ErrLinkNotFound
}

func (f *fakeLinks) Update(_ context.Context, in domain.UpdateLink) (*entities.TenantMember, error) {
	m, ok := f.links[in.ID]
	if !ok {
		return nil, domain.ErrLinkNotFound
	}
	if m.Version != in.Version {
		return nil, errors.New("version mismatch")
	}
	m.Role = in.Role
	m.PrimaryUnitID = in.PrimaryUnitID
	m.Version++
	return m, nil
}

func (f *fakeLinks) Block(_ context.Context, id uuid.UUID) error {
	m, ok := f.links[id]
	if !ok {
		return domain.ErrLinkNotFound
	}
	m.Status = "blocked"
	return nil
}

type fakeEnricher struct {
	codeMap map[string]uuid.UUID // public_code → platform_user_id
	users   map[uuid.UUID]struct {
		Names, LastNames, Email, Code string
	}
}

func (f *fakeEnricher) Hydrate(_ context.Context, members []entities.TenantMember) ([]entities.TenantMember, error) {
	for i, m := range members {
		if u, ok := f.users[m.PlatformUserID]; ok {
			members[i].Names = u.Names
			members[i].LastNames = u.LastNames
			members[i].Email = u.Email
			members[i].PublicCode = u.Code
		}
	}
	return members, nil
}

func (f *fakeEnricher) FindPlatformUserIDByCode(_ context.Context, code string) (uuid.UUID, string, string, string, error) {
	id, ok := f.codeMap[code]
	if !ok {
		return uuid.Nil, "", "", "", domain.ErrPlatformUserNotFound
	}
	u := f.users[id]
	return id, u.Names, u.LastNames, u.Email, nil
}

func TestAddByCode_Success(t *testing.T) {
	links := newFakeLinks()
	uid := uuid.New()
	enricher := &fakeEnricher{
		codeMap: map[string]uuid.UUID{"DEMO-ADMN-0001": uid},
		users: map[uuid.UUID]struct {
			Names, LastNames, Email, Code string
		}{
			uid: {"Ana", "Gomez", "ana@demo.test", "DEMO-ADMN-0001"},
		},
	}
	uc := NewAddByCodeUseCase(AddByCodeDeps{Links: links, Enricher: enricher})

	res, err := uc.Execute(context.Background(), dto.AddMemberRequest{
		PublicCode: "demo-admn-0001",
		Role:       "tenant_admin",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.Names != "Ana" || res.Email != "ana@demo.test" {
		t.Errorf("hydration faltante: %+v", res)
	}
	if len(links.created) != 1 {
		t.Errorf("expected 1 create, got %d", len(links.created))
	}
}

func TestAddByCode_CodeNotFound(t *testing.T) {
	enricher := &fakeEnricher{codeMap: map[string]uuid.UUID{}, users: nil}
	uc := NewAddByCodeUseCase(AddByCodeDeps{Links: newFakeLinks(), Enricher: enricher})

	_, err := uc.Execute(context.Background(), dto.AddMemberRequest{
		PublicCode: "ghost", Role: "resident",
	})
	if !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("expected ErrCodeNotFound, got %v", err)
	}
}

func TestAddByCode_InvalidRole(t *testing.T) {
	uc := NewAddByCodeUseCase(AddByCodeDeps{
		Links:    newFakeLinks(),
		Enricher: &fakeEnricher{},
	})

	_, err := uc.Execute(context.Background(), dto.AddMemberRequest{
		PublicCode: "x", Role: "ghost-role",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAddByCode_AlreadyLinked(t *testing.T) {
	uid := uuid.New()
	enricher := &fakeEnricher{
		codeMap: map[string]uuid.UUID{"DEMO-ADMN-0001": uid},
		users: map[uuid.UUID]struct {
			Names, LastNames, Email, Code string
		}{uid: {"Ana", "Gomez", "ana@demo.test", "DEMO-ADMN-0001"}},
	}
	links := newFakeLinks()
	links.createErr = domain.ErrAlreadyLinked
	uc := NewAddByCodeUseCase(AddByCodeDeps{Links: links, Enricher: enricher})

	_, err := uc.Execute(context.Background(), dto.AddMemberRequest{
		PublicCode: "DEMO-ADMN-0001", Role: "tenant_admin",
	})
	if !errors.Is(err, ErrAlreadyLinked) {
		t.Fatalf("expected ErrAlreadyLinked, got %v", err)
	}
}

func TestList_HydratesAll(t *testing.T) {
	links := newFakeLinks()
	u1 := uuid.New()
	u2 := uuid.New()
	links.links[uuid.New()] = &entities.TenantMember{ID: uuid.New(), PlatformUserID: u1, Role: "guard", Status: "active"}
	links.links[uuid.New()] = &entities.TenantMember{ID: uuid.New(), PlatformUserID: u2, Role: "resident", Status: "active"}

	enricher := &fakeEnricher{
		users: map[uuid.UUID]struct {
			Names, LastNames, Email, Code string
		}{
			u1: {"Pedro", "P", "p@x", "AAAA-0001-0001"},
			u2: {"Lucia", "L", "l@y", "AAAA-0002-0002"},
		},
	}
	uc := NewListUseCase(ListDeps{Links: links, Enricher: enricher})

	res, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	for _, it := range res.Items {
		if it.Names == "" || it.Email == "" {
			t.Errorf("item sin hidratar: %+v", it)
		}
	}
}

func TestUpdate_VersionMismatch(t *testing.T) {
	links := newFakeLinks()
	id := uuid.New()
	links.links[id] = &entities.TenantMember{ID: id, Role: "resident", Status: "active", Version: 1}
	uc := NewUpdateUseCase(UpdateDeps{Links: links, Enricher: &fakeEnricher{}})

	_, err := uc.Execute(context.Background(), id.String(), dto.UpdateMemberRequest{
		Role: "owner", Version: 9,
	})
	if !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("expected ErrVersionMismatch, got %v", err)
	}
}

func TestUpdate_Success(t *testing.T) {
	links := newFakeLinks()
	id := uuid.New()
	links.links[id] = &entities.TenantMember{ID: id, Role: "resident", Status: "active", Version: 1}
	uc := NewUpdateUseCase(UpdateDeps{Links: links, Enricher: &fakeEnricher{}})

	res, err := uc.Execute(context.Background(), id.String(), dto.UpdateMemberRequest{
		Role: "owner", Version: 1,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.Role != "owner" {
		t.Errorf("role no actualizado: %s", res.Role)
	}
}

func TestUpdate_LinkNotFound(t *testing.T) {
	uc := NewUpdateUseCase(UpdateDeps{Links: newFakeLinks(), Enricher: &fakeEnricher{}})

	_, err := uc.Execute(context.Background(), uuid.New().String(), dto.UpdateMemberRequest{
		Role: "owner", Version: 1,
	})
	if !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("expected ErrLinkNotFound, got %v", err)
	}
}

func TestBlock_Success(t *testing.T) {
	links := newFakeLinks()
	id := uuid.New()
	links.links[id] = &entities.TenantMember{ID: id, Status: "active"}
	uc := NewBlockUseCase(BlockDeps{Links: links})

	if err := uc.Execute(context.Background(), id.String()); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if links.links[id].Status != "blocked" {
		t.Errorf("status no actualizado: %s", links.links[id].Status)
	}
}

func TestBlock_BadID(t *testing.T) {
	uc := NewBlockUseCase(BlockDeps{Links: newFakeLinks()})

	if err := uc.Execute(context.Background(), "bogus"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
