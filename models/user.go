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

	db.Preload("Role.Permissions").Offset(offset).Limit(limit).Find(&users)
	return users
}

// Password is excluded: this struct only ever holds the hash.
func (user *User) Validate() error {
	if err := ValidateContact(user.FirstName, user.LastName, user.Email); err != nil {
		return err
	}
	if user.RoleId == 0 {
		return errs.RoleRequired
	}
	return nil
}

// ValidateContact is the one policy for the required, unique, format-checked
// login key: every path that writes firstName/lastName/email must go
// through it, so the column can't disagree with itself about what a valid
// value looks like.
func ValidateContact(firstName, lastName, email string) error {
	if firstName == "" || lastName == "" {
		return errs.MissingField.WithMessage("first and last name are required")
	}
	if email == "" {
		return errs.MissingField.WithMessage("email is required")
	}
	if err := checkmail.ValidateFormat(email); err != nil {
		return errs.EmailInvalid.Wrap(err)
	}
	return nil
}
