package role_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"github.com/loloDawit/go-admin/services/identity/internal/role"
)

// fakeRepository is an in-memory double; each test builds its own, so none depends on another's state.
type fakeRepository struct {
	nextID        int64
	byID          map[int64]role.Role
	byName        map[string]int64
	deleteBlocked map[int64]bool
	// others is CountActiveStaffWithEditStaffOutsideRole's canned answer:
	// role_test has no staff model of its own to derive it from.
	others int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byID:          make(map[int64]role.Role),
		byName:        make(map[string]int64),
		deleteBlocked: make(map[int64]bool),
	}
}

func (f *fakeRepository) Create(_ context.Context, in role.CreateRole) (role.Role, error) {
	if _, taken := f.byName[in.Name]; taken {
		return role.Role{}, role.ErrNameTaken
	}
	f.nextID++
	rl := role.Role{ID: f.nextID, Name: in.Name, Permissions: append([]string{}, in.Permissions...)}
	f.byID[rl.ID] = rl
	f.byName[rl.Name] = rl.ID
	return rl, nil
}

func (f *fakeRepository) Update(_ context.Context, id int64, in role.UpdateRole) (role.Role, error) {
	rl, ok := f.byID[id]
	if !ok {
		return role.Role{}, role.ErrNotFound
	}
	if in.Name != nil {
		if otherID, taken := f.byName[*in.Name]; taken && otherID != id {
			return role.Role{}, role.ErrNameTaken
		}
		delete(f.byName, rl.Name)
		rl.Name = *in.Name
		f.byName[rl.Name] = id
	}
	if in.Permissions != nil {
		rl.Permissions = append([]string{}, (*in.Permissions)...)
	}
	f.byID[id] = rl
	return rl, nil
}

func (f *fakeRepository) Delete(_ context.Context, id int64) error {
	rl, ok := f.byID[id]
	if !ok {
		return role.ErrNotFound
	}
	if f.deleteBlocked[id] {
		return role.ErrInUse
	}
	delete(f.byID, id)
	delete(f.byName, rl.Name)
	return nil
}

func (f *fakeRepository) GetByID(_ context.Context, id int64) (role.Role, error) {
	rl, ok := f.byID[id]
	if !ok {
		return role.Role{}, role.ErrNotFound
	}
	return rl, nil
}

func (f *fakeRepository) List(context.Context) ([]role.Role, error) {
	var out []role.Role
	for _, rl := range f.byID {
		out = append(out, rl)
	}
	return out, nil
}

func (f *fakeRepository) HasEditStaffPermission(_ context.Context, roleID int64) (bool, error) {
	rl, ok := f.byID[roleID]
	if !ok {
		return false, nil
	}
	return slices.Contains(rl.Permissions, string(permission.EditStaff)), nil
}

func (f *fakeRepository) CountActiveStaffWithEditStaffOutsideRole(_ context.Context, _ int64) (int, error) {
	return f.others, nil
}

func newTestService() (*role.Service, *fakeRepository) {
	repo := newFakeRepository()
	return role.NewService(repo), repo
}

func TestCreateRejectsADuplicateName(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	in := role.CreateRole{Name: "manager", Permissions: []string{string(permission.ViewStaff)}}

	if _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.Create(ctx, in); !errors.Is(err, role.ErrNameTaken) {
		t.Fatalf("want ErrNameTaken, got %v", err)
	}
}

func TestCreateRejectsAnUnknownPermission(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Create(context.Background(), role.CreateRole{Name: "ghost", Permissions: []string{"delete_everything"}})
	if !errors.Is(err, role.ErrUnknownPermission) {
		t.Fatalf("want ErrUnknownPermission, got %v", err)
	}
}

func TestUpdateWithNilPermissionsLeavesThemUnchanged(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	created, err := svc.Create(ctx, role.CreateRole{Name: "manager", Permissions: []string{string(permission.ViewStaff)}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	name := "manager-renamed"
	updated, err := svc.Update(ctx, created.ID, role.UpdateRole{Name: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(updated.Permissions) != 1 || updated.Permissions[0] != string(permission.ViewStaff) {
		t.Fatalf("permissions changed by a name-only update: got %v", updated.Permissions)
	}
}

func TestUpdateReplacesPermissionsWhenProvided(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	created, err := svc.Create(ctx, role.CreateRole{Name: "manager", Permissions: []string{string(permission.ViewStaff)}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newPerms := []string{string(permission.ViewOrders), string(permission.EditOrders)}
	updated, err := svc.Update(ctx, created.ID, role.UpdateRole{Permissions: &newPerms})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(updated.Permissions) != 2 {
		t.Fatalf("permissions: want 2, got %v", updated.Permissions)
	}
}

// TestUpdateRefusesToRemoveEditStaffFromTheLastRoleGrantingIt pins the second lockout path: a role update, not only a staff demotion.
func TestUpdateRefusesToRemoveEditStaffFromTheLastRoleGrantingIt(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	created, err := svc.Create(ctx, role.CreateRole{Name: "admin", Permissions: []string{string(permission.EditStaff)}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	repo.others = 0

	empty := []string{}
	if _, err := svc.Update(ctx, created.ID, role.UpdateRole{Permissions: &empty}); !errors.Is(err, role.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin, got %v", err)
	}
}

func TestUpdateAllowsRemovingEditStaffWhenAnotherRoleStillGrantsIt(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	created, err := svc.Create(ctx, role.CreateRole{Name: "admin", Permissions: []string{string(permission.EditStaff)}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	repo.others = 1

	empty := []string{}
	if _, err := svc.Update(ctx, created.ID, role.UpdateRole{Permissions: &empty}); err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestUpdateAllowsUnrelatedPermissionChangesOnTheOnlyAdminRole(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	created, err := svc.Create(ctx, role.CreateRole{Name: "admin", Permissions: []string{string(permission.EditStaff)}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	repo.others = 0

	kept := []string{string(permission.EditStaff), string(permission.ViewOrders)}
	if _, err := svc.Update(ctx, created.ID, role.UpdateRole{Permissions: &kept}); err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestDeleteRefusesARoleAssignedToStaff(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	created, err := svc.Create(ctx, role.CreateRole{Name: "manager"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	repo.deleteBlocked[created.ID] = true

	if err := svc.Delete(ctx, created.ID); !errors.Is(err, role.ErrInUse) {
		t.Fatalf("want ErrInUse, got %v", err)
	}
}

func TestDeleteReturnsNotFoundForAnUnknownRole(t *testing.T) {
	svc, _ := newTestService()
	if err := svc.Delete(context.Background(), 999); !errors.Is(err, role.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
