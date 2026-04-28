package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/authorization/domain"
	"github.com/saas-ph/api/internal/modules/authorization/domain/entities"
)

// fakeRoleRepo es una implementacion en memoria de RoleSetter +
// los subset adicionales que los usecases consumen. Suficiente para
// cubrir golden path + errores principales sin tocar Postgres.
type fakeRoleRepo struct {
	roles       map[string]entities.Role
	rolesByName map[string]string // name -> id
	rolePerms   map[string][]entities.Permission

	failOnCreate         bool
	failOnUpdate         bool
	failOnArchive        bool
	failOnReplaceP       bool
	versionCheckFailures map[string]bool
}

func newFakeRoleRepo() *fakeRoleRepo {
	return &fakeRoleRepo{
		roles:                make(map[string]entities.Role),
		rolesByName:          make(map[string]string),
		rolePerms:            make(map[string][]entities.Permission),
		versionCheckFailures: make(map[string]bool),
	}
}

func (r *fakeRoleRepo) seed(role entities.Role, perms ...entities.Permission) {
	r.roles[role.ID] = role
	r.rolesByName[role.Name] = role.ID
	if len(perms) > 0 {
		r.rolePerms[role.ID] = append([]entities.Permission(nil), perms...)
	}
}

func (r *fakeRoleRepo) ListActive(_ context.Context) ([]entities.Role, error) {
	out := make([]entities.Role, 0, len(r.roles))
	for _, ro := range r.roles {
		if ro.IsActive() {
			out = append(out, ro)
		}
	}
	return out, nil
}

func (r *fakeRoleRepo) GetByID(_ context.Context, id string) (entities.Role, error) {
	role, ok := r.roles[id]
	if !ok || role.DeletedAt != nil {
		return entities.Role{}, domain.ErrRoleNotFound
	}
	return role, nil
}

func (r *fakeRoleRepo) GetByName(_ context.Context, name string) (entities.Role, error) {
	id, ok := r.rolesByName[name]
	if !ok {
		return entities.Role{}, domain.ErrRoleNotFound
	}
	role := r.roles[id]
	if role.DeletedAt != nil {
		return entities.Role{}, domain.ErrRoleNotFound
	}
	return role, nil
}

func (r *fakeRoleRepo) Create(_ context.Context, p domain.CreateRoleParams) (entities.Role, error) {
	if r.failOnCreate {
		return entities.Role{}, errors.New("boom-create")
	}
	if _, exists := r.rolesByName[p.Name]; exists {
		return entities.Role{}, domain.ErrRoleNameTaken
	}
	now := time.Now()
	id := "role-" + p.Name
	role := entities.Role{
		ID:          id,
		Name:        p.Name,
		Description: p.Description,
		IsSystem:    false,
		Status:      entities.StatusActive,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   p.CreatedBy,
		UpdatedBy:   p.CreatedBy,
	}
	r.roles[id] = role
	r.rolesByName[p.Name] = id
	return role, nil
}

func (r *fakeRoleRepo) UpdateName(_ context.Context, p domain.UpdateRoleParams) (entities.Role, error) {
	if r.failOnUpdate {
		return entities.Role{}, errors.New("boom-update")
	}
	role, ok := r.roles[p.ID]
	if !ok {
		return entities.Role{}, domain.ErrRoleNotFound
	}
	if r.versionCheckFailures[p.ID] {
		return entities.Role{}, errors.New("version conflict")
	}
	if role.Version != p.Version {
		return entities.Role{}, errors.New("version mismatch")
	}
	delete(r.rolesByName, role.Name)
	role.Name = p.Name
	role.Description = p.Description
	role.Version++
	role.UpdatedBy = p.UpdatedBy
	role.UpdatedAt = time.Now()
	r.roles[p.ID] = role
	r.rolesByName[role.Name] = role.ID
	return role, nil
}

func (r *fakeRoleRepo) Archive(_ context.Context, id string, by *string) error {
	if r.failOnArchive {
		return errors.New("boom-archive")
	}
	role, ok := r.roles[id]
	if !ok {
		return domain.ErrRoleNotFound
	}
	if role.IsSystem {
		return domain.ErrSystemRoleImmutable
	}
	now := time.Now()
	role.Status = entities.StatusArchived
	role.DeletedAt = &now
	role.DeletedBy = by
	r.roles[id] = role
	delete(r.rolesByName, role.Name)
	return nil
}

func (r *fakeRoleRepo) ListPermissionsForRole(_ context.Context, roleID string) ([]entities.Permission, error) {
	return r.rolePerms[roleID], nil
}

func (r *fakeRoleRepo) AssignPermission(_ context.Context, roleID, permissionID string) error {
	for _, p := range r.rolePerms[roleID] {
		if p.ID == permissionID {
			return nil
		}
	}
	r.rolePerms[roleID] = append(r.rolePerms[roleID], entities.Permission{ID: permissionID, Namespace: permissionID})
	return nil
}

func (r *fakeRoleRepo) RevokePermission(_ context.Context, roleID, permissionID string) error {
	out := make([]entities.Permission, 0, len(r.rolePerms[roleID]))
	for _, p := range r.rolePerms[roleID] {
		if p.ID != permissionID {
			out = append(out, p)
		}
	}
	r.rolePerms[roleID] = out
	return nil
}

func (r *fakeRoleRepo) ReplacePermissions(_ context.Context, roleID string, ids []string) error {
	if r.failOnReplaceP {
		return errors.New("boom-replace")
	}
	out := make([]entities.Permission, 0, len(ids))
	for _, id := range ids {
		out = append(out, entities.Permission{ID: id, Namespace: id, Status: entities.StatusActive})
	}
	r.rolePerms[roleID] = out
	return nil
}

// fakePermRepo es una implementacion en memoria del PermissionRepository.
type fakePermRepo struct {
	perms map[string]entities.Permission // ns -> Permission
}

func newFakePermRepo() *fakePermRepo {
	return &fakePermRepo{perms: make(map[string]entities.Permission)}
}

func (p *fakePermRepo) seed(perm entities.Permission) { p.perms[perm.Namespace] = perm }

func (p *fakePermRepo) List(_ context.Context) ([]entities.Permission, error) {
	out := make([]entities.Permission, 0, len(p.perms))
	for _, pp := range p.perms {
		out = append(out, pp)
	}
	return out, nil
}

func (p *fakePermRepo) GetByNamespace(_ context.Context, ns string) (entities.Permission, error) {
	pp, ok := p.perms[ns]
	if !ok {
		return entities.Permission{}, domain.ErrPermissionNotFound
	}
	return pp, nil
}

// fakeAssignmentRepo en memoria.
type fakeAssignmentRepo struct {
	assignments map[string]entities.RoleAssignment
	byUser      map[string][]string // userID -> []assignmentID
	permsByUser map[string][]string

	failOnCreate bool
	failOnRevoke bool
	failOnList   bool
}

func newFakeAssignmentRepo() *fakeAssignmentRepo {
	return &fakeAssignmentRepo{
		assignments: make(map[string]entities.RoleAssignment),
		byUser:      make(map[string][]string),
		permsByUser: make(map[string][]string),
	}
}

func (r *fakeAssignmentRepo) Create(_ context.Context, p domain.AssignmentParams) (entities.RoleAssignment, error) {
	if r.failOnCreate {
		return entities.RoleAssignment{}, errors.New("boom-create-assign")
	}
	id := "assign-" + p.UserID + "-" + p.RoleID
	if _, exists := r.assignments[id]; exists {
		// solo si esta activa
		if r.assignments[id].IsActive() {
			return entities.RoleAssignment{}, domain.ErrAssignmentDuplicate
		}
	}
	now := time.Now()
	a := entities.RoleAssignment{
		ID:        id,
		UserID:    p.UserID,
		RoleID:    p.RoleID,
		ScopeType: p.ScopeType,
		ScopeID:   p.ScopeID,
		GrantedBy: p.GrantedBy,
		GrantedAt: now,
		Status:    entities.StatusActive,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.assignments[id] = a
	r.byUser[p.UserID] = append(r.byUser[p.UserID], id)
	return a, nil
}

func (r *fakeAssignmentRepo) Revoke(_ context.Context, id string, by *string, reason string) error {
	if r.failOnRevoke {
		return errors.New("boom-revoke")
	}
	a, ok := r.assignments[id]
	if !ok {
		return domain.ErrAssignmentNotFound
	}
	if a.RevokedAt != nil {
		return domain.ErrAssignmentNotFound
	}
	now := time.Now()
	a.RevokedAt = &now
	a.RevocationReason = &reason
	a.Status = "revoked"
	a.UpdatedBy = by
	a.UpdatedAt = now
	r.assignments[id] = a
	return nil
}

func (r *fakeAssignmentRepo) GetActiveByUser(_ context.Context, userID string) ([]entities.RoleAssignment, error) {
	if r.failOnList {
		return nil, errors.New("boom-list")
	}
	ids := r.byUser[userID]
	out := make([]entities.RoleAssignment, 0, len(ids))
	for _, id := range ids {
		if a := r.assignments[id]; a.IsActive() {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeAssignmentRepo) ListPermissionNamespacesForUser(_ context.Context, userID string) ([]string, error) {
	if r.failOnList {
		return nil, errors.New("boom-list-perms")
	}
	return append([]string(nil), r.permsByUser[userID]...), nil
}
