package schemacheck

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUndefinedTableMatchesSQLSTATE42P01(t *testing.T) {
	err := &pgconn.PgError{Code: "42P01"}
	if !isUndefinedTable(err) {
		t.Error("want true for SQLSTATE 42P01 (undefined_table)")
	}
}

func TestIsUndefinedTableRejectsOtherErrors(t *testing.T) {
	if isUndefinedTable(errors.New("connection refused")) {
		t.Error("want false for a non-Postgres error")
	}
	if isUndefinedTable(&pgconn.PgError{Code: "23505"}) {
		t.Error("want false for an unrelated SQLSTATE (unique_violation)")
	}
}
