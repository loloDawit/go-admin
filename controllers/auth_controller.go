package controllers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
	"gitlab.nordstrom.com/go-admin/utils"
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

	user := models.User{
		FirstName: data["firstname"],
		LastName:  data["lastname"],
		Email:     data["email"],
		RoleId:    1,
	}
	user.SetPassword(data["password"])
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

	if err := user.CompareHashAndPassword(data["password"]); err != nil {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"msg": "password|email is incorrect",
		})
	}

	// create user clamis
	token, err := utils.GenerateJWT(strconv.Itoa(user.Id))
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

func User(ctx *fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")

	id, _ := utils.ParseJWT(cookie)

	var user models.User

	database.DB.Where("id = ?", id).First(&user)

	return ctx.JSON(user)
}
