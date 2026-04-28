package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/saas-ph/api/internal/modules/authorization/domain"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

func TestCreateRole_GoldenPath(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()

	uc := CreateRole{Roles: roles}
	out, err := uc.Execute(context.Background(), CreateRoleInput{
		Name:          "junior_admin",
		Description:   "limited admin",
		PermissionIDs: []string{"perm-1", "perm-2"},
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Name != "junior_admin" {
		t.Fatalf("name = %q", out.Name)
	}
	if out.IsSystem {
		t.Fatal("should not be system role")
	}
	if len(out.Permissions) != 2 {
		t.Fatalf("permissions = %d", len(out.Permissions))
	}
}

func TestCreateRole_DuplicateName(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "tenant_admin", Status: entities.StatusActive, IsSystem: true, Version: 1})

	uc := CreateRole{Roles: roles}
	_, err := uc.Execute(context.Background(), CreateRoleInput{Name: "tenant_admin"})
	if !errors.Is(err, domain.ErrRoleNameTaken) {
		t.Fatalf("err = %v; want ErrRoleNameTaken", err)
	}
}

func TestCreateRole_RequiresName(t *testing.T) {
	t.Parallel()
	uc := CreateRole{Roles: newFakeRoleRepo()}
	_, err := uc.Execute(context.Background(), CreateRoleInput{Name: "   "})
	if err == nil {
		t.Fatal("expected error on empty name")
	}
}

func TestUpdateRole_GoldenPath(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "audit_lite", Status: entities.StatusActive, Version: 1})

	uc := UpdateRole{Roles: roles}
	pids := []string{"perm-x"}
	out, err := uc.Execute(context.Background(), UpdateRoleInput{
		ID:            "r-1",
		Name:          "audit_extended",
		Description:   "now broader",
		PermissionIDs: &pids,
		Version:       1,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Name != "audit_extended" {
		t.Fatalf("name = %q", out.Name)
	}
	if len(out.Permissions) != 1 || out.Permissions[0].ID != "perm-x" {
		t.Fatalf("permissions = %+v", out.Permissions)
	}
}

func TestUpdateRole_BlockSystem(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "tenant_admin", Status: entities.StatusActive, IsSystem: true, Version: 1})

	uc := UpdateRole{Roles: roles}
	_, err := uc.Execute(context.Background(), UpdateRoleInput{ID: "r-1", Name: "new_name", Version: 1})
	if !errors.Is(err, domain.ErrSystemRoleImmutable) {
		t.Fatalf("err = %v; want ErrSystemRoleImmutable", err)
	}
}

func TestUpdateRole_NotFound(t *testing.T) {
	t.Parallel()
	uc := UpdateRole{Roles: newFakeRoleRepo()}
	_, err := uc.Execute(context.Background(), UpdateRoleInput{ID: "r-missing", Name: "x", Version: 1})
	if !errors.Is(err, domain.ErrRoleNotFound) {
		t.Fatalf("err = %v; want ErrRoleNotFound", err)
	}
}

func TestUpdateRole_NameClash(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "audit", Status: entities.StatusActive, Version: 1})
	roles.seed(entities.Role{ID: "r-2", Name: "audit_full", Status: entities.StatusActive, Version: 1})

	uc := UpdateRole{Roles: roles}
	_, err := uc.Execute(context.Background(), UpdateRoleInput{ID: "r-1", Name: "audit_full", Version: 1})
	if !errors.Is(err, domain.ErrRoleNameTaken) {
		t.Fatalf("err = %v; want ErrRoleNameTaken", err)
	}
}

func TestDeleteRole_GoldenPath(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "audit", Status: entities.StatusActive, Version: 1})

	uc := DeleteRole{Roles: roles}
	if err := uc.Execute(context.Background(), DeleteRoleInput{ID: "r-1"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if _, err := roles.GetByID(context.Background(), "r-1"); !errors.Is(err, domain.ErrRoleNotFound) {
		t.Fatalf("expected role to be soft-deleted; got %v", err)
	}
}

func TestDeleteRole_BlockSystem(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "tenant_admin", Status: entities.StatusActive, IsSystem: true, Version: 1})

	uc := DeleteRole{Roles: roles}
	err := uc.Execute(context.Background(), DeleteRoleInput{ID: "r-1"})
	if !errors.Is(err, domain.ErrSystemRoleImmutable) {
		t.Fatalf("err = %v; want ErrSystemRoleImmutable", err)
	}
}

func TestListRoles(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "admin", Status: entities.StatusActive, Version: 1})
	roles.seed(entities.Role{ID: "r-2", Name: "guard", Status: entities.StatusActive, Version: 1})

	uc := ListRoles{Roles: roles}
	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(out))
	}
}

func TestGetRole_WithPermissions(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(
		entities.Role{ID: "r-1", Name: "owner", Status: entities.StatusActive, Version: 1},
		entities.Permission{ID: "p-1", Namespace: "package.read"},
	)

	uc := GetRole{Roles: roles}
	out, err := uc.Execute(context.Background(), "r-1")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(out.Permissions) != 1 || out.Permissions[0].Namespace != "package.read" {
		t.Fatalf("permissions = %+v", out.Permissions)
	}
}

func TestGetRole_NotFound(t *testing.T) {
	t.Parallel()
	uc := GetRole{Roles: newFakeRoleRepo()}
	_, err := uc.Execute(context.Background(), "missing")
	if !errors.Is(err, domain.ErrRoleNotFound) {
		t.Fatalf("err = %v; want ErrRoleNotFound", err)
	}
}
