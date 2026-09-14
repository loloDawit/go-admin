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
