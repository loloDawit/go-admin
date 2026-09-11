package models

import (
	"github.com/badoux/checkmail"
	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/errs"
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

func (user *User) SetPassword(h auth.Hasher, plain string) error {
	hashed, err := h.Hash(plain)
	if err != nil {
		return err
	}
	user.Password = hashed
	return nil
}

// Cost is read from the stored hash, so any Hasher verifies any hash.
func (user *User) CompareHashAndPassword(plain string) error {
	return auth.Hasher{}.Check(user.Password, plain)
}

func (user *User) Count(db *gorm.DB) int64 {
	var total int64
	db.Model(&User{}).Count(&total)

	return total

}

func (user *User) Take(db *gorm.DB, limit int, offset int) interface{} {
	var users []User

	db.Preload("Role").Offset(offset).Limit(limit).Find(&users)
	return users
}

// Password is excluded: this struct only ever holds the hash.
func (user *User) Validate() error {
	if user.FirstName == "" || user.LastName == "" {
		return errs.NameRequired
	}
	if user.Email == "" {
		return errs.EmailRequired
	}
	if err := checkmail.ValidateFormat(user.Email); err != nil {
		return errs.EmailInvalid.Wrap(err)
	}
	return nil
}
