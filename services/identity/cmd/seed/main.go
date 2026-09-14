// Command seed creates the first owner account. It is the only way an account
// comes into existence before staff management exists: there is no public
// registration endpoint, because self-registration granted admin rights.
package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/loloDawit/go-admin/services/identity/internal/errs"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
)

const adminRoleName = "admin"

type config struct {
	databaseURL string
	email       string
	password    string
	bcryptCost  int
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := run(ctx, cfg)
	if err != nil {
		logger.Error("seed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if created {
		logger.Info("owner created", slog.String("email", cfg.email))
		return
	}
	logger.Info("owner already exists, left unchanged", slog.String("email", cfg.email))
}

func loadConfig() (config, error) {
	cfg := config{
		databaseURL: os.Getenv("DATABASE_URL"),
		email:       os.Getenv("OWNER_EMAIL"),
		password:    os.Getenv("OWNER_PASSWORD"),
	}
	if cfg.databaseURL == "" {
		return config{}, errs.ErrMissingDatabaseURL
	}
	if cfg.email == "" {
		return config{}, errs.ErrMissingOwnerEmail
	}
	if cfg.password == "" {
		return config{}, errs.ErrMissingOwnerPassword
	}

	cost, err := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	if err != nil {
		return config{}, errs.Wrap(errs.OpReadBcryptCost, err)
	}
	cfg.bcryptCost = cost
	return cfg, nil
}

func run(ctx context.Context, cfg config) (bool, error) {
	conn, err := pgx.Connect(ctx, cfg.databaseURL)
	if err != nil {
		return false, errs.Wrap(errs.OpConnectDatabase, err)
	}
	defer conn.Close(ctx)

	var roles int
	if err := conn.QueryRow(ctx, countAdminRoleStmt, adminRoleName).Scan(&roles); err != nil {
		return false, errs.Wrap(errs.OpCountAdminRole, err)
	}
	if roles == 0 {
		return false, errs.ErrNoAdminRole
	}

	hash, err := session.NewHasher(cfg.bcryptCost).Hash(cfg.password)
	if err != nil {
		return false, errs.Wrap(errs.OpHashOwnerPassword, err)
	}

	tag, err := conn.Exec(ctx, insertOwnerStmt, cfg.email, "Owner", "Account", hash, adminRoleName)
	if err != nil {
		return false, errs.Wrap(errs.OpInsertOwner, err)
	}
	return tag.RowsAffected() == 1, nil
}
