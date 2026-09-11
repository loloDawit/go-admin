package database

import (
	"fmt"

	"github.com/loloDawit/go-admin/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB is a package-level global. This is a known wart inherited from the
// original code: it prevents dependency injection and forces tests to assign
// to it directly. M1 replaces it with an injected store. Do not build new
// code that depends on it being global.
var DB *gorm.DB

// Connect opens a connection using the supplied DSN and runs AutoMigrate.
// It returns an error instead of panicking so main can report it cleanly.
//
// AutoMigrate is itself a known wart (ASSESSMENT 4w) — it cannot be reviewed,
// rolled back, or ordered. M1 replaces it with golang-migrate. M0 keeps it so
// that this milestone changes no schema semantics.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}

	DB = db
	return db, nil
}

// Migrate applies the schema. Exported so the test harness can build a
// database without going through Connect's global assignment.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	return nil
}
