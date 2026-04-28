package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/saas-ph/api/internal/modules/authorization/domain"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

func ptr(s string) *string { return &s }

func TestAssignRole_GoldenPath(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "guard", Status: entities.StatusActive, Version: 1})
	assigns := newFakeAssignmentRepo()

	uc := AssignRole{Roles: roles, Assignments: assigns}
	towerType := entities.ScopeTower
	towerID := "tower-1"
	out, err := uc.Execute(context.Background(), AssignRoleInput{
		UserID:    "u-1",
		RoleID:    "r-1",
		ScopeType: &towerType,
		ScopeID:   &towerID,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.UserID != "u-1" || out.RoleID != "r-1" {
		t.Fatalf("got %+v", out)
	}
}

func TestAssignRole_RoleNotFound(t *testing.T) {
	t.Parallel()
	uc := AssignRole{Roles: newFakeRoleRepo(), Assignments: newFakeAssignmentRepo()}
	_, err := uc.Execute(context.Background(), AssignRoleInput{UserID: "u-1", RoleID: "missing"})
	if !errors.Is(err, domain.ErrRoleNotFound) {
		t.Fatalf("err = %v; want ErrRoleNotFound", err)
	}
}

func TestAssignRole_DuplicateAssignment(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "guard", Status: entities.StatusActive, Version: 1})
	assigns := newFakeAssignmentRepo()

	uc := AssignRole{Roles: roles, Assignments: assigns}
	if _, err := uc.Execute(context.Background(), AssignRoleInput{UserID: "u-1", RoleID: "r-1"}); err != nil {
		t.Fatalf("first assign: %v", err)
	}
	if _, err := uc.Execute(context.Background(), AssignRoleInput{UserID: "u-1", RoleID: "r-1"}); !errors.Is(err, domain.ErrAssignmentDuplicate) {
		t.Fatalf("err = %v; want ErrAssignmentDuplicate", err)
	}
}

func TestAssignRole_InvalidScope(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "guard", Status: entities.StatusActive, Version: 1})
	assigns := newFakeAssignmentRepo()

	uc := AssignRole{Roles: roles, Assignments: assigns}

	// scope_id without scope_type
	_, err := uc.Execute(context.Background(), AssignRoleInput{
		UserID: "u-1", RoleID: "r-1", ScopeID: ptr("x"),
	})
	if err == nil {
		t.Fatal("expected error for orphan scope_id")
	}

	// tower without scope_id
	towerType := entities.ScopeTower
	_, err = uc.Execute(context.Background(), AssignRoleInput{
		UserID: "u-1", RoleID: "r-1", ScopeType: &towerType,
	})
	if err == nil {
		t.Fatal("expected error for tower scope without id")
	}

	// tenant with scope_id
	tenantType := entities.ScopeTenant
	_, err = uc.Execute(context.Background(), AssignRoleInput{
		UserID: "u-1", RoleID: "r-1", ScopeType: &tenantType, ScopeID: ptr("x"),
	})
	if err == nil {
		t.Fatal("expected error for tenant scope with id")
	}

	// invalid type
	bad := "stage"
	_, err = uc.Execute(context.Background(), AssignRoleInput{
		UserID: "u-1", RoleID: "r-1", ScopeType: &bad, ScopeID: ptr("x"),
	})
	if err == nil {
		t.Fatal("expected error for unsupported scope_type")
	}
}

func TestUnassignRole_GoldenPath(t *testing.T) {
	t.Parallel()
	roles := newFakeRoleRepo()
	roles.seed(entities.Role{ID: "r-1", Name: "guard", Status: entities.StatusActive, Version: 1})
	assigns := newFakeAssignmentRepo()

	if _, err := (AssignRole{Roles: roles, Assignments: assigns}).Execute(context.Background(),
		AssignRoleInput{UserID: "u-1", RoleID: "r-1"}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	uc := UnassignRole{Assignments: assigns}
	if err := uc.Execute(context.Background(), UnassignRoleInput{UserID: "u-1", RoleID: "r-1", Reason: "left"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestUnassignRole_NotFound(t *testing.T) {
	t.Parallel()
	uc := UnassignRole{Assignments: newFakeAssignmentRepo()}
	err := uc.Execute(context.Background(), UnassignRoleInput{UserID: "u-1", RoleID: "r-1"})
	if !errors.Is(err, domain.ErrAssignmentNotFound) {
		t.Fatalf("err = %v; want ErrAssignmentNotFound", err)
	}
}

func TestResolveUserPermissions(t *testing.T) {
	t.Parallel()
	assigns := newFakeAssignmentRepo()
	assigns.permsByUser["u-1"] = []string{"package.read", "visit.read"}

	uc := ResolveUserPermissions{Assignments: assigns}
	out, err := uc.Execute(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(out.Permissions) != 2 {
		t.Fatalf("perms = %v", out.Permissions)
	}
}

func TestResolveUserPermissions_EmptyUserID(t *testing.T) {
	t.Parallel()
	uc := ResolveUserPermissions{Assignments: newFakeAssignmentRepo()}
	_, err := uc.Execute(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected error on empty user_id")
	}
}

func TestResolveUserPermissions_NoPermissionsReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	uc := ResolveUserPermissions{Assignments: newFakeAssignmentRepo()}
	out, err := uc.Execute(context.Background(), "u-empty")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Permissions == nil || len(out.Permissions) != 0 {
		t.Fatalf("expected empty (non-nil) slice; got %#v", out.Permissions)
	}
}
