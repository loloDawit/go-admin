package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect() {
	dns := "test-user:password@/go_admin"
	_, err := gorm.Open(mysql.Open(dns), &gorm.Config{})

	if err != nil {
		panic("Could not connect to the database")
	}

}
