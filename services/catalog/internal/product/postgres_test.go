package product

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSortColumnIsAnAllowlistNotAString(t *testing.T) {
	if _, _, err := resolveSort("title;DROP TABLE products"); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort for an injected sort value, got %v", err)
	}
	if _, _, err := resolveSort("unknown_column"); !errors.Is(err, ErrInvalidSort) {
		t.Fatalf("want ErrInvalidSort for a column absent from the allowlist, got %v", err)
	}
}

func TestResolveSortMapsKnownKeysToTheirColumn(t *testing.T) {
	col, desc, err := resolveSort("title")
	if err != nil || col != "title" || desc {
		t.Fatalf("title: got col=%q desc=%v err=%v", col, desc, err)
	}
	col, desc, err = resolveSort("-price")
	if err != nil || col != "price_minor" || !desc {
		t.Fatalf("-price: got col=%q desc=%v err=%v", col, desc, err)
	}
}

func TestResolveSortDefaultsToNewestFirst(t *testing.T) {
	col, desc, err := resolveSort("")
	if err != nil || col != "created_at" || !desc {
		t.Fatalf("default: got col=%q desc=%v err=%v", col, desc, err)
	}
}

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
