package models

type User struct {
	Id        int
	FirstName string
	LastName  string
	Email     string `gorm:"unique"`
	Password  []byte
}
