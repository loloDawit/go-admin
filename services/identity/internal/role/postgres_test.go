package role

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsNoRowsMatchesPgxErrNoRows(t *testing.T) {
	if !isNoRows(pgx.ErrNoRows) {
		t.Error("want true for pgx.ErrNoRows")
	}
	if isNoRows(errors.New("connection refused")) {
		t.Error("want false for a non-pgx.ErrNoRows error")
	}
}

func TestIsUniqueViolationMatchesOnlyCode23505(t *testing.T) {
	if !isUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Error("want true for SQLSTATE 23505")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Error("want false for a different SQLSTATE (foreign key violation)")
	}
	if isUniqueViolation(nil) {
		t.Error("want false for a nil error")
	}
}

func TestIsForeignKeyViolationMatchesOnlyCode23503(t *testing.T) {
	if !isForeignKeyViolation(&pgconn.PgError{Code: "23503"}) {
		t.Error("want true for SQLSTATE 23503")
	}
	if isForeignKeyViolation(&pgconn.PgError{Code: "23505"}) {
		t.Error("want false for a different SQLSTATE (unique violation)")
	}
	if isForeignKeyViolation(nil) {
		t.Error("want false for a nil error")
	}
}

// fakeExecer stands in for *pgxpool.Pool/pgx.Tx with a caller-set affected count.
type fakeExecer struct {
	affected int64
}

func (f fakeExecer) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 " + strconv.FormatInt(f.affected, 10)), nil
}

// TestInsertRolePermissionsCatchesANameAbsentFromPermissions pins a fix: a name matching no row inserts nothing and raises no SQL error on its own.
func TestInsertRolePermissionsCatchesANameAbsentFromPermissions(t *testing.T) {
	e := fakeExecer{affected: 1} // one of two names matched no row
	err := insertRolePermissions(context.Background(), e, 1, []string{"view_staff", "not_a_real_permission"})
	if !errors.Is(err, ErrPermissionRowMismatch) {
		t.Fatalf("want ErrPermissionRowMismatch, got %v", err)
	}
}

func TestInsertRolePermissionsSucceedsWhenEveryNameMatchesARow(t *testing.T) {
	e := fakeExecer{affected: 2}
	if err := insertRolePermissions(context.Background(), e, 1, []string{"view_staff", "edit_staff"}); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func TestInsertRolePermissionsIsANoOpForNoNames(t *testing.T) {
	e := fakeExecer{affected: 0}
	if err := insertRolePermissions(context.Background(), e, 1, nil); err != nil {
		t.Fatalf("insert: %v", err)
	}
}
