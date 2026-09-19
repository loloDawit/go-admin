//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustInt64 converts one of the string IDs the API answers with back to the
// bigint pool.Exec/QueryRow need: the driver does not implicitly cast a text
// parameter against a bigint column.
func mustInt64(t *testing.T, s string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatalf("id %q is not an integer: %v", s, err)
	}
	return id
}

// These mirror staff/dto.go and role/dto.go's JSON shapes; test/integration
// cannot import identity's internal packages, so the shapes are duplicated
// here rather than reused.
type staffResponse struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	FirstName          string `json:"firstName"`
	LastName           string `json:"lastName"`
	RoleID             string `json:"roleId"`
	IsActive           bool   `json:"isActive"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

type createStaffResponse struct {
	Staff    staffResponse `json:"staff"`
	Password string        `json:"password"`
}

// Paged, like every other list: items plus the effective page, size and total.
type listStaffResponse struct {
	Items    []staffResponse `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
	Total    int             `json:"total"`
}

type roleResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	MemberCount int      `json:"memberCount"`
}

type listRoleResponse struct {
	Items    []roleResponse `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
}

type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// apiCall drives one request through the gateway as c and decodes both a
// possible success body (into out, when non-nil) and the error envelope
// (harmless to attempt against a success body: unmatched fields are ignored).
func apiCall(t *testing.T, c *http.Client, method, path string, body, out any) (int, errorEnvelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, gatewayURL()+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if len(raw) > 0 {
		if out != nil {
			_ = json.Unmarshal(raw, out)
		}
		var envelope errorEnvelope
		_ = json.Unmarshal(raw, &envelope)
		return resp.StatusCode, envelope
	}
	return resp.StatusCode, errorEnvelope{}
}

func identityPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestStaffLifecycleThroughTheGateway drives create, list, get, a partial
// patch, and deactivate against the real routes, as the seeded owner.
func TestStaffLifecycleThroughTheGateway(t *testing.T) {
	pool := identityPool(t)
	c := loggedInClient(t)

	roleName := fmt.Sprintf("integration-role-%d", time.Now().UnixNano())
	var role roleResponse
	status, _ := apiCall(t, c, http.MethodPost, "/api/v1/roles",
		map[string]any{"name": roleName, "permissions": []string{"view_staff"}}, &role)
	if status != http.StatusCreated {
		t.Fatalf("create role: want 201, got %d", status)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM roles WHERE id = $1`, mustInt64(t, role.ID))
	})

	email := fmt.Sprintf("integration-staff-%d@example.com", time.Now().UnixNano())
	var created createStaffResponse
	status, _ = apiCall(t, c, http.MethodPost, "/api/v1/staff", map[string]any{
		"email": email, "firstName": "Integ", "lastName": "Ration", "roleId": role.ID,
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create staff: want 201, got %d", status)
	}
	if created.Password == "" {
		t.Error("create staff: want a one-time password, got none")
	}
	if !created.Staff.MustChangePassword {
		t.Error("a newly created staff member must be flagged to change their password")
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM staff WHERE id = $1`, mustInt64(t, created.Staff.ID))
	})

	var got staffResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/staff/"+created.Staff.ID, nil, &got); status != http.StatusOK {
		t.Fatalf("get staff: want 200, got %d", status)
	}
	if got.Email != email {
		t.Errorf("get staff: email: want %q, got %q", email, got.Email)
	}

	var list listStaffResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/staff", nil, &list); status != http.StatusOK {
		t.Fatalf("list staff: want 200, got %d", status)
	}
	found := false
	for _, s := range list.Items {
		if s.ID == created.Staff.ID {
			found = true
		}
	}
	if !found {
		t.Error("list staff: the created staff member is missing")
	}

	var patched staffResponse
	status, _ = apiCall(t, c, http.MethodPatch, "/api/v1/staff/"+created.Staff.ID,
		map[string]any{"lastName": "Patched"}, &patched)
	if status != http.StatusOK {
		t.Fatalf("patch staff: want 200, got %d", status)
	}
	if patched.LastName != "Patched" {
		t.Errorf("patch staff: lastName: want Patched, got %q", patched.LastName)
	}
	if patched.FirstName != "Integ" {
		t.Errorf("patch staff: an untouched field changed: firstName want Integ, got %q", patched.FirstName)
	}

	var deactivated staffResponse
	status, _ = apiCall(t, c, http.MethodPost, "/api/v1/staff/"+created.Staff.ID+"/deactivate", nil, &deactivated)
	if status != http.StatusOK {
		t.Fatalf("deactivate staff: want 200, got %d", status)
	}
	if deactivated.IsActive {
		t.Error("deactivate staff: isActive is still true")
	}
}

// TestCreateStaffWithAnUnknownRoleIdIsRejected pins the fix for the foreign
// key violation staff/postgres.go used to leave unmapped: this must be a
// client mistake (422), never an unmapped 500.
func TestCreateStaffWithAnUnknownRoleIdIsRejected(t *testing.T) {
	c := loggedInClient(t)

	email := fmt.Sprintf("integration-badrole-%d@example.com", time.Now().UnixNano())
	status, body := apiCall(t, c, http.MethodPost, "/api/v1/staff", map[string]any{
		"email": email, "firstName": "No", "lastName": "Role", "roleId": "99999999",
	}, nil)
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("create staff with a nonexistent roleId: want 422, got %d", status)
	}
	if body.Code != "validation_failed" {
		t.Errorf("code: want validation_failed, got %q", body.Code)
	}
}

// TestRoleLifecycleThroughTheGateway drives create, list, get, patch name,
// patch permissions, and delete against the real routes.
func TestRoleLifecycleThroughTheGateway(t *testing.T) {
	pool := identityPool(t)
	c := loggedInClient(t)

	name := fmt.Sprintf("integration-role-%d", time.Now().UnixNano())
	var created roleResponse
	status, _ := apiCall(t, c, http.MethodPost, "/api/v1/roles",
		map[string]any{"name": name, "permissions": []string{"view_staff", "view_roles"}}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create role: want 201, got %d", status)
	}
	roleCleanup := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM roles WHERE id = $1`, mustInt64(t, created.ID))
	}
	t.Cleanup(roleCleanup)

	var got roleResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/roles/"+created.ID, nil, &got); status != http.StatusOK {
		t.Fatalf("get role: want 200, got %d", status)
	}

	var list listRoleResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/roles", nil, &list); status != http.StatusOK {
		t.Fatalf("list roles: want 200, got %d", status)
	}
	found := false
	for _, r := range list.Items {
		if r.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Error("list roles: the created role is missing")
	}

	renamed := name + "-renamed"
	var patchedName roleResponse
	status, _ = apiCall(t, c, http.MethodPatch, "/api/v1/roles/"+created.ID,
		map[string]any{"name": renamed}, &patchedName)
	if status != http.StatusOK {
		t.Fatalf("patch role name: want 200, got %d", status)
	}
	if patchedName.Name != renamed {
		t.Errorf("patch role name: want %q, got %q", renamed, patchedName.Name)
	}

	var patchedPerms roleResponse
	status, _ = apiCall(t, c, http.MethodPatch, "/api/v1/roles/"+created.ID,
		map[string]any{"permissions": []string{"view_orders"}}, &patchedPerms)
	if status != http.StatusOK {
		t.Fatalf("patch role permissions: want 200, got %d", status)
	}
	if len(patchedPerms.Permissions) != 1 || patchedPerms.Permissions[0] != "view_orders" {
		t.Errorf("patch role permissions: want [view_orders], got %v", patchedPerms.Permissions)
	}

	// A role assigned to staff must be refused, not deleted out from under them.
	email := fmt.Sprintf("integration-roleuser-%d@example.com", time.Now().UnixNano())
	var assignee createStaffResponse
	if status, _ := apiCall(t, c, http.MethodPost, "/api/v1/staff", map[string]any{
		"email": email, "firstName": "Role", "lastName": "User", "roleId": created.ID,
	}, &assignee); status != http.StatusCreated {
		t.Fatalf("create assignee: want 201, got %d", status)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM staff WHERE id = $1`, mustInt64(t, assignee.Staff.ID))
	})

	status, body := apiCall(t, c, http.MethodDelete, "/api/v1/roles/"+created.ID, nil, nil)
	if status != http.StatusConflict {
		t.Fatalf("delete an in-use role: want 409, got %d", status)
	}
	if body.Code != "role_in_use" {
		t.Errorf("code: want role_in_use, got %q", body.Code)
	}

	// Now unassign and delete for real, proving the happy path once the role is free.
	if _, err := pool.Exec(context.Background(), `DELETE FROM staff WHERE id = $1`, mustInt64(t, assignee.Staff.ID)); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	if status, _ := apiCall(t, c, http.MethodDelete, "/api/v1/roles/"+created.ID, nil, nil); status != http.StatusNoContent {
		t.Fatalf("delete role: want 204, got %d", status)
	}
	// roleCleanup still runs on t.Cleanup; against an already-deleted row it is a harmless no-op.
}

// TestDeactivatingTheOnlyActiveAdminIsRefused exercises the staff-side
// last-admin guard through POST .../deactivate, which (unlike the general
// PATCH route) never redirects a caller acting on their own id through the
// self-update path — so the owner can trip this guard on themselves.
func TestDeactivatingTheOnlyActiveAdminIsRefused(t *testing.T) {
	ctx := context.Background()
	pool := identityPool(t)
	ownerEmail := required(t, "OWNER_EMAIL")

	// Registered before the risky call: if the guard is broken, this repairs
	// the very state a broken guard would otherwise leave behind.
	var parked []int64
	rows, err := pool.Query(ctx, `
		UPDATE staff SET is_active = false
		WHERE is_active AND email <> $1
		RETURNING id`, ownerEmail)
	if err != nil {
		t.Fatalf("park other admins: %v", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("scan parked: %v", err)
		}
		parked = append(parked, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("park other admins: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE staff SET is_active = true WHERE id = ANY($1)`, parked)
	})
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE staff SET is_active = true WHERE email = $1`, ownerEmail)
	})

	c := loggedInClient(t)
	var me meResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/me", nil, &me); status != http.StatusOK {
		t.Fatalf("me: want 200, got %d", status)
	}

	status, body := apiCall(t, c, http.MethodPost, "/api/v1/staff/"+me.StaffID+"/deactivate", nil, nil)
	if status != http.StatusConflict {
		t.Fatalf("deactivate the only admin: want 409, got %d", status)
	}
	if body.Code != "last_admin" {
		t.Errorf("code: want last_admin, got %q", body.Code)
	}

	var stillActive bool
	if err := pool.QueryRow(ctx, `SELECT is_active FROM staff WHERE email = $1`, ownerEmail).Scan(&stillActive); err != nil {
		t.Fatalf("check owner is_active: %v", err)
	}
	if !stillActive {
		t.Fatal("the guard did not fire: the owner is now inactive")
	}
}

// TestRemovingEditStaffFromTheOnlyAdminRoleIsRefused exercises the role-side
// last-admin guard: editing the role every current admin belongs to, down to
// no edit_staff, must be refused rather than lock the system out.
func TestRemovingEditStaffFromTheOnlyAdminRoleIsRefused(t *testing.T) {
	ctx := context.Background()
	pool := identityPool(t)
	ownerEmail := required(t, "OWNER_EMAIL")
	c := loggedInClient(t)

	var ownerRoleID string
	var currentPerms []string
	var me meResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/me", nil, &me); status != http.StatusOK {
		t.Fatalf("me: want 200, got %d", status)
	}
	var got staffResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/staff/"+me.StaffID, nil, &got); status != http.StatusOK {
		t.Fatalf("get owner: want 200, got %d", status)
	}
	ownerRoleID = got.RoleID
	ownerRoleIDInt := mustInt64(t, ownerRoleID)

	var role roleResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/roles/"+ownerRoleID, nil, &role); status != http.StatusOK {
		t.Fatalf("get owner's role: want 200, got %d", status)
	}
	currentPerms = role.Permissions

	// Park every active staff member outside the owner's role: otherwise
	// another role holding edit_staff would make the removal legitimate.
	var parked []int64
	rows, err := pool.Query(ctx, `
		UPDATE staff SET is_active = false
		WHERE is_active AND role_id <> $1
		RETURNING id`, ownerRoleIDInt)
	if err != nil {
		t.Fatalf("park staff outside the role: %v", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("scan parked: %v", err)
		}
		parked = append(parked, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("park staff outside the role: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE staff SET is_active = true WHERE id = ANY($1)`, parked)
	})

	// Restored regardless of outcome: if the guard is broken, this call
	// really does strip edit_staff from the role the owner logs in with.
	t.Cleanup(func() {
		var perms []string
		_, _ = pool.Exec(context.Background(), `DELETE FROM role_permissions WHERE role_id = $1`, ownerRoleIDInt)
		for _, p := range currentPerms {
			perms = append(perms, p)
		}
		_, _ = pool.Exec(context.Background(), `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT $1, id FROM permissions WHERE name = ANY($2::text[])`, ownerRoleIDInt, perms)
	})
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE staff SET is_active = true WHERE email = $1`, ownerEmail)
	})

	var withoutEditStaff []string
	for _, p := range currentPerms {
		if p != "edit_staff" {
			withoutEditStaff = append(withoutEditStaff, p)
		}
	}

	status, body := apiCall(t, c, http.MethodPatch, "/api/v1/roles/"+ownerRoleID,
		map[string]any{"permissions": withoutEditStaff}, nil)
	if status != http.StatusConflict {
		t.Fatalf("remove edit_staff from the only admin role: want 409, got %d", status)
	}
	if body.Code != "last_admin" {
		t.Errorf("code: want last_admin, got %q", body.Code)
	}

	var stillHasEditStaff bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id = $1 AND p.name = 'edit_staff'
	)`, ownerRoleIDInt).Scan(&stillHasEditStaff); err != nil {
		t.Fatalf("check role still has edit_staff: %v", err)
	}
	if !stillHasEditStaff {
		t.Fatal("the guard did not fire: the owner's role lost edit_staff")
	}
}

// Two roles each grant edit_staff to one active staff member; both are demoted
// at once. Row locks alone do not stop this: the role guard locks staff rows
// while the write changes role_permissions, so neither transaction sees the
// other and both leave the system with nobody able to manage staff.
func TestConcurrentRoleDemotionsCannotBothSucceed(t *testing.T) {
	ctx := context.Background()
	pool := identityPool(t)

	var parked []int64
	rows, err := pool.Query(ctx, `UPDATE staff SET is_active = false WHERE is_active RETURNING id`)
	if err != nil {
		t.Fatalf("park existing: %v", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("scan parked: %v", err)
		}
		parked = append(parked, id)
	}
	rows.Close()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE staff SET is_active = true WHERE id = ANY($1)`, parked)
	})

	roleIDs := make([]int64, 2)
	for i := range roleIDs {
		name := fmt.Sprintf("demote-race-%d-%d", time.Now().UnixNano(), i)
		if err := pool.QueryRow(ctx, `INSERT INTO roles (name) VALUES ($1) RETURNING id`, name).Scan(&roleIDs[i]); err != nil {
			t.Fatalf("insert role: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_id)
			SELECT $1, id FROM permissions WHERE name = 'edit_staff'`, roleIDs[i]); err != nil {
			t.Fatalf("grant edit_staff: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO staff (email, first_name, last_name, password_hash, role_id, is_active)
			VALUES ($1, 'D', 'R', 'x', $2, true)`, name+"@example.com", roleIDs[i]); err != nil {
			t.Fatalf("insert staff: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM staff WHERE role_id = ANY($1)`, roleIDs)
		_, _ = pool.Exec(context.Background(), `DELETE FROM roles WHERE id = ANY($1)`, roleIDs)
	})

	held := make(chan struct{})
	starting := make(chan struct{})
	results := make(chan error, 2)

	go func() {
		results <- demoteRoleGuarded(ctx, pool, roleIDs[0], func() {
			close(held)
			<-starting
			time.Sleep(300 * time.Millisecond)
		})
	}()
	go func() {
		<-held
		close(starting)
		results <- demoteRoleGuarded(ctx, pool, roleIDs[1], func() {})
	}()

	succeeded := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("want exactly 1 role demotion to succeed, got %d", succeeded)
	}

	var remaining int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM staff s
		JOIN role_permissions rp ON rp.role_id = s.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE p.name = 'edit_staff' AND s.is_active`).Scan(&remaining); err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if remaining == 0 {
		t.Fatal("no active staff can manage staff: the system locked itself out")
	}
}

var errWouldBeLastAdmin = errors.New("would remove the last edit_staff holder")

func demoteRoleGuarded(ctx context.Context, pool *pgxpool.Pool, roleID int64, afterLock func()) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(4812001)`); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT s.role_id FROM staff s
		JOIN role_permissions rp ON rp.role_id = s.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE p.name = 'edit_staff' AND s.is_active
		FOR UPDATE OF s`)
	if err != nil {
		return err
	}
	others := 0
	for rows.Next() {
		var other int64
		if err := rows.Scan(&other); err != nil {
			rows.Close()
			return err
		}
		if other != roleID {
			others++
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	afterLock()
	if others == 0 {
		return errWouldBeLastAdmin
	}

	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions
		WHERE role_id = $1 AND permission_id = (SELECT id FROM permissions WHERE name = 'edit_staff')`, roleID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
