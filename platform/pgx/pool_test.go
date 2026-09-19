package pgx_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	pgxplatform "github.com/loloDawit/go-admin/platform/pgx"
)

func TestNewPoolRejectsAnEmptyDSN(t *testing.T) {
	_, err := pgxplatform.NewPool(context.Background(), "")
	if !errors.Is(err, pgxplatform.ErrEmptyDSN) {
		t.Fatalf("want ErrEmptyDSN, got %v", err)
	}
}

// A malformed DSN must fail at construction, not at first query.
func TestNewPoolRejectsAMalformedDSN(t *testing.T) {
	_, err := pgxplatform.NewPool(context.Background(), "postgres://u:p@db.internal:notaport/x")
	if err == nil {
		t.Fatal("a malformed DSN must be rejected")
	}
	if strings.Contains(err.Error(), "db.internal") {
		t.Error("the error must not echo the DSN's host")
	}
	if strings.Contains(err.Error(), "password") {
		t.Error("the error must not echo credential material")
	}
}

// The tracer is what turns every query in every service into a span. Without
// it "which dependency caused the latency?" has no answer for the database,
// which is the dependency most often responsible.
func TestPoolHasAQueryTracer(t *testing.T) {
	pool, err := pgxplatform.NewPool(context.Background(), "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	if pool.Config().ConnConfig.Tracer == nil {
		t.Fatal("ConnConfig.Tracer is nil")
	}
}
