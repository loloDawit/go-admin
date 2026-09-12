package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func dsn(t *testing.T, user, password, database string) string {
	t.Helper()
	host := os.Getenv("PGHOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "5433"
	}
	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + database + "?sslmode=disable"
}

func TestServiceRoleCannotReachAnotherServiceDatabase(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "catalog_db"))
	if err == nil {
		conn.Close(ctx)
		t.Fatal("identity_user must not be able to connect to catalog_db")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		t.Fatalf("want a permission error, got: %v", err)
	}
}

func TestServiceRoleCannotReachAnotherServiceDatabaseCrossCheck(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "orders_user", "dev_only_orders", "identity_db"))
	if err == nil {
		conn.Close(ctx)
		t.Fatal("orders_user must not be able to connect to identity_db")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		t.Fatalf("want a permission error, got: %v", err)
	}
}

func TestServiceRoleReachesItsOwnDatabase(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("identity_user must reach identity_db: %v", err)
	}
	defer conn.Close(ctx)

	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("query: %v", err)
	}
}

// Grants are only meaningful if the role cannot simply grant itself more.
func TestServiceRoleHasNoElevatedAttributes(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	var super, createdb, createrole bool
	err = conn.QueryRow(ctx,
		`SELECT rolsuper, rolcreatedb, rolcreaterole FROM pg_roles WHERE rolname = current_user`,
	).Scan(&super, &createdb, &createrole)
	if err != nil {
		t.Fatalf("query role attributes: %v", err)
	}

	if super {
		t.Error("service role must be NOSUPERUSER")
	}
	if createdb {
		t.Error("service role must be NOCREATEDB")
	}
	if createrole {
		t.Error("service role must be NOCREATEROLE")
	}
}
