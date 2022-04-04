package models

import (
	"errors"
	"github.com/badoux/checkmail"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	Id        int    `json:"id"`
	FirstName string `json:"firstName" gorm:"size:255;not null"`
	LastName  string `json:"lastName" gorm:"size:255;not null"`
	Email     string `json:"email" gorm:"size:100;not null;unique"`
	Password  string `json:"-"`
	RoleId    uint   `json:"roleId"`
	Role      Role   `json:"role" gorm:"foreignKey:RoleId"`
}

func (user *User) SetPassword(password string) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test"), 14)
	user.Password = string(hashedPassword)
}

func (user *User) CompareHashAndPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

func (user *User) Count(db *gorm.DB) int64 {
	var total int64
	db.Model(&User{}).Count(&total)

	return total

}

func (user *User) Take(db *gorm.DB, limit int, offset int) interface{} {
	var users []User

	db.Preload("role").Offset(offset).Limit(limit).Find(&users)
	return users
}

func (user *User) Validate(action string) error {
	if user.FirstName == "" || user.LastName == "" {
		return errors.New("first and last name is required")
	}
	if user.Password == "" {
		return errors.New("required password")
	}
	if user.Email == "" {
		return errors.New("required email")
	}
	if err := checkmail.ValidateFormat(user.Email); err != nil {
		return errors.New("invalid email")
	}
	return nil
}
