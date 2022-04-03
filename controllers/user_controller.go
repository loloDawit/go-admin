package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllUsers(ctx *fiber.Ctx) error {
	var users []models.User

	database.DB.Preload("Role").Find(&users)

	return ctx.JSON(users)
}

func GetUser(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	database.DB.Preload("Role").Find(&user)

	return ctx.JSON(user)
}

func UpdateUser(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	if err := ctx.BodyParser(&user); err != nil {
		return err
	}

	database.DB.Model(&user).Updates(user)

	return ctx.JSON(user)
}

func DeleteUser(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	database.DB.Delete(&user)

	return ctx.JSON(fiber.Map{
		"msg": "success",
	})
}

func CreateUser(ctx *fiber.Ctx) error {
	var user models.User

	if err := ctx.BodyParser(&user); err != nil {
		return err
	}

	user.SetPassword("124")

	database.DB.Create(&user)
	return ctx.JSON(user)
}
