package staff_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
	"github.com/loloDawit/go-admin/services/identity/internal/staff"
)

const testCost = 4

// fakeRepository is an in-memory double; each test builds its own so none
// touches a database or depends on another test's state.
type fakeRepository struct {
	nextID      int64
	byID        map[int64]record
	editStaffOf map[int64]bool
	revokedFor  []int64
}

type record struct {
	staff.Staff
	passwordHash string
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: make(map[int64]record), editStaffOf: map[int64]bool{1: true}}
}

// addRole keys the guard on the permission alone, not the role's name.
func (f *fakeRepository) addRole(id int64) {
	f.editStaffOf[id] = true
}

func (f *fakeRepository) Create(_ context.Context, in staff.CreateStaff, passwordHash string) (staff.Staff, error) {
	for _, r := range f.byID {
		if r.Email == in.Email {
			return staff.Staff{}, staff.ErrEmailTaken
		}
	}
	f.nextID++
	st := staff.Staff{
		ID: f.nextID, Email: in.Email, FirstName: in.FirstName, LastName: in.LastName,
		RoleID: in.RoleID, IsActive: true, MustChangePassword: true,
	}
	f.byID[st.ID] = record{Staff: st, passwordHash: passwordHash}
	return st, nil
}

func (f *fakeRepository) Update(_ context.Context, id int64, in staff.UpdateStaff) (staff.Staff, error) {
	r, ok := f.byID[id]
	if !ok {
		return staff.Staff{}, staff.ErrNotFound
	}
	if in.Email != nil {
		for otherID, other := range f.byID {
			if otherID != id && other.Email == *in.Email {
				return staff.Staff{}, staff.ErrEmailTaken
			}
		}
		r.Email = *in.Email
	}
	if in.FirstName != nil {
		r.FirstName = *in.FirstName
	}
	if in.LastName != nil {
		r.LastName = *in.LastName
	}
	if in.RoleID != nil {
		r.RoleID = *in.RoleID
	}
	if in.IsActive != nil {
		r.IsActive = *in.IsActive
	}
	f.byID[id] = r
	return r.Staff, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id int64) (staff.Staff, error) {
	r, ok := f.byID[id]
	if !ok {
		return staff.Staff{}, staff.ErrNotFound
	}
	return r.Staff, nil
}

func (f *fakeRepository) List(context.Context) ([]staff.Staff, error) {
	var out []staff.Staff
	for _, r := range f.byID {
		out = append(out, r.Staff)
	}
	return out, nil
}

func (f *fakeRepository) PasswordHash(_ context.Context, id int64) (string, error) {
	r, ok := f.byID[id]
	if !ok {
		return "", staff.ErrNotFound
	}
	return r.passwordHash, nil
}

func (f *fakeRepository) SetPasswordHash(_ context.Context, id int64, hash string) error {
	r, ok := f.byID[id]
	if !ok {
		return staff.ErrNotFound
	}
	r.passwordHash = hash
	r.MustChangePassword = false
	f.byID[id] = r
	return nil
}

func (f *fakeRepository) HasEditStaffPermission(_ context.Context, roleID int64) (bool, error) {
	return f.editStaffOf[roleID], nil
}

func (f *fakeRepository) LockActiveEditStaffExcluding(ctx context.Context, excludeID int64) (int, error) {
	return f.CountOtherActiveStaffWithEditStaff(ctx, excludeID)
}

func (f *fakeRepository) RevokeSessions(_ context.Context, staffID int64) error {
	f.revokedFor = append(f.revokedFor, staffID)
	return nil
}

func (f *fakeRepository) RunInTx(ctx context.Context, fn func(staff.Repository) error) error {
	return fn(f)
}

func (f *fakeRepository) CountOtherActiveStaffWithEditStaff(_ context.Context, excludeID int64) (int, error) {
	count := 0
	for id, r := range f.byID {
		if id != excludeID && r.IsActive && f.editStaffOf[r.RoleID] {
			count++
		}
	}
	return count, nil
}

func newTestService() (*staff.Service, *fakeRepository) {
	repo := newFakeRepository()
	return staff.NewService(repo, session.NewHasher(testCost)), repo
}

func TestCreateReturnsAPasswordThatWorksExactlyOnce(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	st, pw, err := svc.Create(ctx, staff.CreateStaff{Email: "a@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if pw == "" {
		t.Fatal("no password returned")
	}

	got, err := svc.Get(ctx, st.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.Contains(fmt.Sprint(got), pw) {
		t.Fatal("the generated password is readable after creation")
	}
}

func TestACreatedStaffMemberMustChangeTheirPassword(t *testing.T) {
	svc, _ := newTestService()
	st, _, err := svc.Create(context.Background(), staff.CreateStaff{Email: "a@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !st.MustChangePassword {
		t.Fatal("want must_change_password set on a newly created staff member")
	}
}

func TestCreateRejectsADuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	in := staff.CreateStaff{Email: "dup@example.com", FirstName: "A", LastName: "B", RoleID: 1}

	if _, _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, _, err := svc.Create(ctx, in); !errors.Is(err, staff.ErrEmailTaken) {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}
}

// Pins the service half of the self-update guard; handler_test.go's TestSelfUpdateCannotChangeRole pins the HTTP half.
func TestUpdateLeavesRoleUnchangedWithoutARoleIDField(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	st, _, err := svc.Create(ctx, staff.CreateStaff{Email: "self@example.com", FirstName: "A", LastName: "B", RoleID: 2})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// RoleID and IsActive stay nil, as a self-update DTO with no roleId field would leave them.
	name := "Changed"
	updated, err := svc.Update(ctx, st.ID, staff.UpdateStaff{FirstName: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.RoleID != 2 {
		t.Fatalf("role changed via self-update: want 2, got %d", updated.RoleID)
	}
}

func TestDeactivateRefusesToRemoveTheLastActiveAdmin(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	admin, _, err := svc.Create(ctx, staff.CreateStaff{Email: "admin@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.Deactivate(ctx, admin.ID); !errors.Is(err, staff.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin, got %v", err)
	}
}

func TestUpdateWithOnlyNameFieldsDoesNotTripTheLastAdminGuard(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	admin, _, err := svc.Create(ctx, staff.CreateStaff{Email: "admin@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	name := "Changed"
	updated, err := svc.Update(ctx, admin.ID, staff.UpdateStaff{FirstName: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.IsActive {
		t.Fatal("want the sole admin still active after a name-only update")
	}
}

// The guard must key on permission.EditStaff, not a role literally named "admin".
func TestDeactivateProtectsTheLastEditStaffHolderRegardlessOfRoleName(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	const managerRoleID = 5
	repo.addRole(managerRoleID)

	manager, _, err := svc.Create(ctx, staff.CreateStaff{Email: "manager@example.com", FirstName: "A", LastName: "B", RoleID: managerRoleID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.Deactivate(ctx, manager.ID); !errors.Is(err, staff.ErrLastAdmin) {
		t.Fatalf("want ErrLastAdmin for the last %s holder under a non-admin role name, got %v", permission.EditStaff, err)
	}
}

func TestDeactivateSucceedsWhenAnotherActiveAdminRemains(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	first, _, err := svc.Create(ctx, staff.CreateStaff{Email: "one@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, _, err := svc.Create(ctx, staff.CreateStaff{Email: "two@example.com", FirstName: "A", LastName: "B", RoleID: 1}); err != nil {
		t.Fatalf("create: %v", err)
	}

	deactivated, err := svc.Deactivate(ctx, first.ID)
	if err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if deactivated.IsActive {
		t.Fatal("want is_active false after deactivation")
	}
}

func TestChangePasswordRejectsAWrongCurrentPassword(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	st, _, err := svc.Create(ctx, staff.CreateStaff{Email: "a@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = repo // repo kept for clarity that this test relies on the fake, not a live DB

	err = svc.ChangePassword(ctx, st.ID, "wrong-current-password", "a-new-password")
	if err == nil {
		t.Fatal("want an error for a wrong current password")
	}
}

func TestChangePasswordClearsMustChangePassword(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	st, pw, err := svc.Create(ctx, staff.CreateStaff{Email: "a@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.ChangePassword(ctx, st.ID, pw, "a-new-password"); err != nil {
		t.Fatalf("change password: %v", err)
	}

	got, err := svc.Get(ctx, st.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.MustChangePassword {
		t.Fatal("want must_change_password cleared after a successful change")
	}
}

func TestDeactivationRevokesLiveSessions(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	created, _, err := svc.Create(ctx, staff.CreateStaff{Email: "revoke@example.com", FirstName: "R", LastName: "S", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// A second admin, so the last-admin guard does not refuse the deactivation.
	if _, _, err := svc.Create(ctx, staff.CreateStaff{Email: "other@example.com", FirstName: "O", LastName: "T", RoleID: 1}); err != nil {
		t.Fatalf("create second: %v", err)
	}

	if _, err := svc.Deactivate(ctx, created.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	for _, id := range repo.revokedFor {
		if id == created.ID {
			return
		}
	}
	t.Fatalf("deactivation left live sessions: revoked %v, want %d", repo.revokedFor, created.ID)
}
