//go:build integration

package integration_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

// openAPIDoc reaches components.schemas.Permission.enum; yaml.v3 ignores any
// key with no matching struct field, so the rest of identity.yaml is not
// modeled here.
type openAPIDoc struct {
	Components struct {
		Schemas struct {
			Permission struct {
				Enum []string `yaml:"enum"`
			} `yaml:"Permission"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

// wantPermissions reads the published contract (identity.yaml) rather than
// importing services/identity/internal/permission: that package is internal
// to services/identity and Go's internal-package rule forbids this package,
// rooted outside that tree, from importing it. Reading the YAML also checks
// the more relevant direction: identity.yaml is what Catalog and Orders
// generate constants from in M3/M4, not permission.All() itself.
func wantPermissions() []string {
	data, err := os.ReadFile("../../services/identity/openapi/identity.yaml")
	if err != nil {
		panic(err)
	}
	var doc openAPIDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		panic(err)
	}
	return doc.Components.Schemas.Permission.Enum
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
