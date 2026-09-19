package customer

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
		t.Error("want false for a different SQLSTATE")
	}
	if isUniqueViolation(nil) {
		t.Error("want false for a nil error")
	}
}

// A sort column cannot be a bind parameter, so this is the one place
// injection is structurally possible: anything absent from the allowlist
// must be refused rather than interpolated into the ORDER BY clause.
func TestCustomerSortColumnIsAnAllowlistNotAString(t *testing.T) {
	if _, _, err := resolveCustomerSort("name; DROP TABLE customers"); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort for an unknown sort key, got %v", err)
	}

	col, desc, err := resolveCustomerSort("email")
	if err != nil {
		t.Fatalf("resolveCustomerSort(email): %v", err)
	}
	if col != "email" || desc {
		t.Fatalf("got col=%q desc=%v, want email ascending", col, desc)
	}
}

func TestCustomerSortColumnPrefixMeansDescending(t *testing.T) {
	col, desc, err := resolveCustomerSort("-created_at")
	if err != nil {
		t.Fatalf("resolveCustomerSort(-created_at): %v", err)
	}
	if col != "created_at" || !desc {
		t.Fatalf("got col=%q desc=%v, want created_at descending", col, desc)
	}
}

func TestOffsetComputesFromPageAndPageSize(t *testing.T) {
	if got := offset(1, 20); got != 0 {
		t.Fatalf("page 1: want offset 0, got %d", got)
	}
	if got := offset(3, 20); got != 40 {
		t.Fatalf("page 3: want offset 40, got %d", got)
	}
}
