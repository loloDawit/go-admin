package session

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestIsNoRowsMatchesPgxErrNoRows(t *testing.T) {
	if !isNoRows(pgx.ErrNoRows) {
		t.Error("want true for pgx.ErrNoRows")
	}
}

func TestIsNoRowsRejectsOtherErrors(t *testing.T) {
	if isNoRows(errors.New("connection refused")) {
		t.Error("want false for a non-pgx.ErrNoRows error")
	}
	if isNoRows(nil) {
		t.Error("want false for a nil error")
	}
}
