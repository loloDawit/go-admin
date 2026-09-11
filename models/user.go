package models

import (
	"errors"
	"github.com/badoux/checkmail"
	"github.com/loloDawit/go-admin/internal/auth"
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

// SetPassword hashes plain at the hasher's configured cost and stores it.
// The Hasher is an explicit parameter so the cost is always a visible
// dependency rather than a package-level constant.
//
// The previous implementation discarded bcrypt's error AND ignored its
// argument, hashing the literal string "test" for every user — so every
// account shared one password (ASSESSMENT 4a).
func (user *User) SetPassword(h auth.Hasher, plain string) error {
	hashed, err := h.Hash(plain)
	if err != nil {
		return err
	}
	user.Password = hashed
	return nil
}

func (user *User) CompareHashAndPassword(plain string) error {
	// Verification reads the cost from the stored hash, so any Hasher works —
	// including for hashes written when the configured cost was different.
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
