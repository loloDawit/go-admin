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

// A sort column cannot be a bind parameter, so this is the one place
// injection is structurally possible: anything absent from the allowlist
// must be refused rather than interpolated into the ORDER BY clause.
func TestRoleSortColumnIsAnAllowlistNotAString(t *testing.T) {
	if _, _, err := resolveRoleSort("name; DROP TABLE roles"); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort for an unknown sort key, got %v", err)
	}

	col, desc, err := resolveRoleSort("members")
	if err != nil {
		t.Fatalf("resolveRoleSort(members): %v", err)
	}
	if col != "member_count" || desc {
		t.Fatalf("got col=%q desc=%v, want member_count ascending", col, desc)
	}
}

func TestRoleSortColumnPrefixMeansDescending(t *testing.T) {
	col, desc, err := resolveRoleSort("-name")
	if err != nil {
		t.Fatalf("resolveRoleSort(-name): %v", err)
	}
	if col != "r.name" || !desc {
		t.Fatalf("got col=%q desc=%v, want r.name descending", col, desc)
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
