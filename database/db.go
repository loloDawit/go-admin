package database

import (
	"gitlab.nordstrom.com/go-admin/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect() {
	dns := "test-user:password@/go_admin"
	db, err := gorm.Open(mysql.Open(dns), &gorm.Config{})

	if err != nil {
		panic("Could not connect to the database")
	}

	db.AutoMigrate(&models.User{})
}
