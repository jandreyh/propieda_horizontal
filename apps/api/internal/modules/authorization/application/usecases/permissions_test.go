package usecases

import (
	"context"
	"testing"

	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

func TestListPermissions(t *testing.T) {
	t.Parallel()
	repo := newFakePermRepo()
	repo.seed(entities.Permission{ID: "p-1", Namespace: "package.read", Status: entities.StatusActive})
	repo.seed(entities.Permission{ID: "p-2", Namespace: "visit.read", Status: entities.StatusActive})

	uc := ListPermissions{Permissions: repo}
	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(out))
	}
}
