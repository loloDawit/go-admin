// Package pgx constructs PostgreSQL connection pools.
package pgx

import (
	"context"
	"errors"
	"time"

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
		// pgx redacts the password itself, but the error still echoes the
		// DSN's host and path; neither may reach a client, so the parse
		// cause is discarded rather than wrapped into the returned error.
		return nil, ErrInvalidDSN
	}

	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	// NewWithConfig does not dial. A database that is down must not stop the
	// process from starting; readiness reports it instead.
	return pgxpool.NewWithConfig(ctx, cfg)
}
