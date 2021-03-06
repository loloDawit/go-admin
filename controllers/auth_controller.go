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

	if data["password"] != data["passwordConfirm"] {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"error": "password do not match",
		})
	}

	user := models.User{
		FirstName: data["firstName"],
		LastName:  data["lastName"],
		Email:     data["email"],
		Password:  data["password"],
		RoleId:    1,
	}

	err := user.Validate("")
	if err != nil {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	user.SetPassword(data["password"])
	if result := database.DB.Create(&user); result.Error != nil {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{
			"error": result.Error,
		})
	}

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
	// Set cookie
	ctx.Cookie(&cookie)

	return ctx.JSON(fiber.Map{
		"token": token,
	})

}

func Logout(ctx *fiber.Ctx) error {
	cookie := fiber.Cookie{
		Name:  "jwt",
		Value: "",
		// Set expiry date to the past
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

func UpdateUserInfo(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	cookie := ctx.Cookies("jwt")
	id, _ := utils.ParseJWT(cookie)

	userId, _ := strconv.Atoi(id)

	user := models.User{
		Id:        userId,
		FirstName: data["firstname"],
		LastName:  data["lastname"],
		Email:     data["email"],
	}

	database.DB.Model(&user).Where("id=?", id).Updates(user)

	return ctx.JSON(user)
}

func UpdatePassword(ctx *fiber.Ctx) error {
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

	cookie := ctx.Cookies("jwt")
	id, _ := utils.ParseJWT(cookie)
	userId, _ := strconv.Atoi(id)
	user := models.User{
		Id: userId,
	}
	user.SetPassword(data["password"])

	database.DB.Model(&user).Where("id=?", id).Updates(user)

	return ctx.JSON(user)
}
