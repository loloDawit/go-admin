package models

import "golang.org/x/crypto/bcrypt"

type User struct {
	Id        int    `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email" gorm:"unique"`
	Password  []byte `json:"-"`
	RoleId    uint   `json:"roleid"`
	Role      Role   `json:"role" gorm:"foreignKey:RoleId"`
}

func (user *User) SetPassword(password string) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test"), 14)
	user.Password = hashedPassword
}

func (user *User) CompareHashAndPassword(password string) error {
	return bcrypt.CompareHashAndPassword(user.Password, []byte(password))
}
