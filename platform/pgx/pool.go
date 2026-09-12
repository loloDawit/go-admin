// Package pgx constructs PostgreSQL connection pools.
package pgx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmptyDSN = errors.New("database DSN is empty")

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, ErrEmptyDSN
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// pgx redacts the password itself, but the error still echoes the
		// DSN's host and path; neither may reach a client.
		return nil, fmt.Errorf("parse database DSN: invalid format")
	}

	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	// NewWithConfig does not dial. A database that is down must not stop the
	// process from starting; readiness reports it instead.
	return pgxpool.NewWithConfig(ctx, cfg)
}
