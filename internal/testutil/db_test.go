package testutil

import (
	"testing"

	"github.com/loloDawit/go-admin/models"
)

func TestNewDBIsMigratedAndEmpty(t *testing.T) {
	db := NewDB(t)

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("users table should exist and be queryable: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected an empty users table, got %d rows", count)
	}
}

func TestNewDBTruncatesBetweenTests(t *testing.T) {
	db := NewDB(t)

	if err := db.Create(&models.Permission{Name: "view_users"}).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	// A second NewDB in the same process must wipe the row above.
	db = NewDB(t)

	var count int64
	db.Model(&models.Permission{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected truncation between NewDB calls, got %d rows", count)
	}
}
