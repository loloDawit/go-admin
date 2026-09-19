// Package pgx constructs PostgreSQL connection pools.
package pgx

import (
	"context"
	"errors"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmptyDSN   = errors.New("database DSN is empty")
	ErrInvalidDSN = errors.New("parse database DSN: invalid format")
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, ErrEmptyDSN
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// The parse error echoes the DSN's host and path, so it is discarded rather than wrapped.
		return nil, ErrInvalidDSN
	}

	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	// Attached here rather than per service, so every query in every service is
	// a span. The span name is the operation word, not the statement: otelpgx
	// keeps the SQL in db.query.text, where redaction rules apply.
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	// NewWithConfig does not dial. A database that is down must not stop the
	// process from starting; readiness reports it instead.
	return pgxpool.NewWithConfig(ctx, cfg)
}
