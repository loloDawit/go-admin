//go:build integration

package integration_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The literal eight stands in for permission.All(), which Task 6 introduces;
// this test cannot import that package before it exists.
func wantPermissions() []string {
	return []string{
		"view_staff", "edit_staff",
		"view_roles", "edit_roles",
		"view_products", "edit_products",
		"view_orders", "edit_orders",
	}
}

func TestSeededPermissionsMatchConstants(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `SELECT name FROM permissions ORDER BY name`)
	if err != nil {
		t.Fatalf("query permissions: %v", err)
	}

	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	want := wantPermissions()

	// SQL ORDER BY and Go's literal order need not agree on collation;
	// sort both independently so the comparison checks set equality.
	slices.Sort(got)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("seeded permissions differ from constants:\n db: %v\n go: %v", got, want)
	}
}

func TestSeededAdminRoleHoldsAllPermissions(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	var count int
	err = conn.QueryRow(ctx, `
		SELECT count(*)
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		WHERE r.name = 'admin'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("query role_permissions: %v", err)
	}

	if want := len(wantPermissions()); count != want {
		t.Fatalf("admin role permission count: want %d, got %d", want, count)
	}
}
