package controllers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
	"golang.org/x/crypto/bcrypt"
)

func Register(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	if data["password"] != data["passwordconfirm"] {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"msg": "password do not match",
		})
	}
	password, err := bcrypt.GenerateFromPassword([]byte(data["password"]), 14)

	if err != nil {
		return err
	}

	user := models.User{
		FirstName: data["firstname"],
		LastName:  data["lastname"],
		Email:     data["email"],
		Password:  password,
	}

	database.DB.Create(&user)

	return ctx.JSON(user)
}

func Login(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	var user models.User

	database.DB.Where("email = ? ", data["email"]).First(&user)

	if user.Id == 0 {
		ctx.Status(404)
		return ctx.JSON(fiber.Map{
			"msg": "user not found",
		})
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data["password"])); err != nil {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"msg": "password|email is incorrect",
		})
	}

	// create user clamis
	clamis := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    strconv.Itoa(user.Id),
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), //24hr
	})

	token, err := clamis.SignedString([]byte("secret"))
	if err != nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}

	ctx.Cookie(&cookie)

	return ctx.JSON(fiber.Map{
		"token": token,
	})

}

func Logout(ctx *fiber.Ctx) error {
	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}

	ctx.Cookie(&cookie)

	return ctx.JSON(fiber.Map{
		"msg": "success",
	})

}

type Clamis struct {
	jwt.StandardClaims
}

func User(ctx *fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")

	token, err := jwt.ParseWithClaims(cookie, &Clamis{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})

	if err != nil || !token.Valid {
		ctx.Status(fiber.StatusUnauthorized)
		return ctx.JSON(fiber.Map{
			"msg": "unauthorized.",
		})
	}

	clamis := token.Claims.(*Clamis)

	var user models.User

	database.DB.Where("id = ?", clamis.Issuer).First(&user)

	return ctx.JSON(user)
}
