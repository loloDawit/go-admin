//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// This matches the SQLSTATE, not message text, which would depend on lc_messages and upstream wording.
func assertConnectRefused(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("want a refused connection, got none")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("want a *pgconn.PgError, got: %v", err)
	}
	if pgErr.Code != "42501" {
		t.Fatalf("want SQLSTATE 42501 (insufficient_privilege), got %s: %v", pgErr.Code, err)
	}
}

func TestServiceRoleCannotReachAnotherServiceDatabase(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "catalog_db"))
	if err == nil {
		conn.Close(ctx)
		t.Fatal("identity_user must not be able to connect to catalog_db")
	}
	assertConnectRefused(t, err)
}

func TestServiceRoleCannotReachAnotherServiceDatabaseCrossCheck(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "orders_user", "dev_only_orders", "identity_db"))
	if err == nil {
		conn.Close(ctx)
		t.Fatal("orders_user must not be able to connect to identity_db")
	}
	assertConnectRefused(t, err)
}

// pg_database and pg_roles are readable to anyone who can connect, so
// leaving postgres/template1 open leaks every service's name and owner.
func TestServiceRoleCannotReachMaintenanceDatabase(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "postgres"))
	if err == nil {
		conn.Close(ctx)
		t.Fatal("identity_user must not be able to connect to postgres")
	}
	assertConnectRefused(t, err)
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

// Grants are only meaningful if a role cannot simply grant itself more, or
// reach another service's data through a path other than CONNECT.
func TestServiceRoleHasNoElevatedAttributes(t *testing.T) {
	roles := []struct {
		user     string
		password string
		database string
	}{
		{"identity_user", "dev_only_identity", "identity_db"},
		{"catalog_user", "dev_only_catalog", "catalog_db"},
		{"orders_user", "dev_only_orders", "orders_db"},
	}

	for _, r := range roles {
		t.Run(r.user, func(t *testing.T) {
			ctx := context.Background()

			conn, err := pgx.Connect(ctx, dsn(t, r.user, r.password, r.database))
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer conn.Close(ctx)

			var super, createdb, createrole, replication bool
			err = conn.QueryRow(ctx,
				`SELECT rolsuper, rolcreatedb, rolcreaterole, rolreplication FROM pg_roles WHERE rolname = current_user`,
			).Scan(&super, &createdb, &createrole, &replication)
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
			// REPLICATION grants pg_basebackup over the whole cluster,
			// bypassing per-database CONNECT checks entirely.
			if replication {
				t.Error("service role must not have REPLICATION")
			}

			var canReadServerFiles bool
			err = conn.QueryRow(ctx,
				`SELECT pg_has_role(current_user, 'pg_read_server_files', 'MEMBER')`,
			).Scan(&canReadServerFiles)
			if err != nil {
				t.Fatalf("query pg_read_server_files membership: %v", err)
			}
			// pg_read_file() and server-side COPY under this role would
			// reach every database's files under $PGDATA/base directly.
			if canReadServerFiles {
				t.Error("service role must not be a member of pg_read_server_files")
			}
		})
	}
}
