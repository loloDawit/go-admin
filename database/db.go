package database

import (
	"fmt"

	"github.com/loloDawit/go-admin/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Deprecated: global state, replaced by an injected store in M1. Do not add
// new code that depends on it.
var DB *gorm.DB

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

// Migrate is exported so the test harness can build a database without
// Connect's global assignment. AutoMigrate is replaced by golang-migrate in M1.
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
