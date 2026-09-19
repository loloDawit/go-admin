package staff

import (
	"errors"
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
	if isUniqueViolation(errors.New("connection refused")) {
		t.Error("want false for a non-pgconn.PgError")
	}
	if isUniqueViolation(nil) {
		t.Error("want false for a nil error")
	}
}

// A sort column cannot be a bind parameter, so this is the one place
// injection is structurally possible: anything absent from the allowlist
// must be refused rather than interpolated into the ORDER BY clause.
func TestStaffSortColumnIsAnAllowlistNotAString(t *testing.T) {
	if _, _, err := resolveStaffSort("email; DROP TABLE staff"); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort for an unknown sort key, got %v", err)
	}

	col, desc, err := resolveStaffSort("email")
	if err != nil {
		t.Fatalf("resolveStaffSort(email): %v", err)
	}
	if col != "email" || desc {
		t.Fatalf("got col=%q desc=%v, want email ascending", col, desc)
	}
}

func TestStaffSortColumnPrefixMeansDescending(t *testing.T) {
	col, desc, err := resolveStaffSort("-created")
	if err != nil {
		t.Fatalf("resolveStaffSort(-created): %v", err)
	}
	if col != "created_at" || !desc {
		t.Fatalf("got col=%q desc=%v, want created_at descending", col, desc)
	}
}

// name sorts by the whole displayed name as one key, so ASC/DESC applies to
// last name and first name together rather than only to last_name.
func TestStaffNameSortIsOneCombinedExpression(t *testing.T) {
	col, _, err := resolveStaffSort("name")
	if err != nil {
		t.Fatalf("resolveStaffSort(name): %v", err)
	}
	if col != "lower(last_name || ' ' || first_name)" {
		t.Fatalf("got col=%q, want a single combined expression", col)
	}
}

func TestIsForeignKeyViolationMatchesOnlyCode23503(t *testing.T) {
	if !isForeignKeyViolation(&pgconn.PgError{Code: "23503"}) {
		t.Error("want true for SQLSTATE 23503")
	}
	if isForeignKeyViolation(&pgconn.PgError{Code: "23505"}) {
		t.Error("want false for a different SQLSTATE (unique violation)")
	}
	if isForeignKeyViolation(errors.New("connection refused")) {
		t.Error("want false for a non-pgconn.PgError")
	}
	if isForeignKeyViolation(nil) {
		t.Error("want false for a nil error")
	}
}
