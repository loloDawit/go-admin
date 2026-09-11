// Package testutil provides integration-test infrastructure. It spins a real
// MySQL container so tests exercise the same driver, dialect, and constraint
// behaviour as production — sqlite-in-memory would hide exactly the GORM
// bugs this milestone exists to fix.
package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/database"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once     sync.Once
	sharedDB *gorm.DB
	initErr  error
)

// NewDB returns a migrated, empty database. The container is started once per
// `go test` process and reused; each call truncates every table so tests do
// not leak state into one another.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	once.Do(func() {
		ctx := context.Background()

		container, err := tcmysql.Run(ctx, "mysql:8.4",
			tcmysql.WithDatabase("go_admin_test"),
			tcmysql.WithUsername("test"),
			tcmysql.WithPassword("test"),
		)
		if err != nil {
			initErr = err
			return
		}

		dsn, err := container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4", "loc=UTC")
		if err != nil {
			initErr = err
			return
		}

		// The container reports ready before MySQL accepts connections;
		// retry briefly rather than failing the whole suite on a cold start.
		var db *gorm.DB
		for i := 0; i < 30; i++ {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			initErr = err
			return
		}

		if err := database.Migrate(db); err != nil {
			initErr = err
			return
		}
		sharedDB = db
	})

	if initErr != nil {
		t.Fatalf("testutil: could not start MySQL container: %v", initErr)
	}

	truncateAll(t, sharedDB)

	// M0 wart: controllers read the package-level global. Assign it so
	// handlers under test hit the container. Removed in M1.
	database.DB = sharedDB

	return sharedDB
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()

	tables := []string{
		"role_permissions", "order_items", "orders",
		"products", "users", "roles", "permissions",
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		t.Fatalf("testutil: disable FK checks: %v", err)
	}
	for _, table := range tables {
		if err := db.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatalf("testutil: truncate %s: %v", table, err)
		}
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		t.Fatalf("testutil: re-enable FK checks: %v", err)
	}
}
