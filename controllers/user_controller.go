package controllers

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllUsers(ctx *fiber.Ctx) error {
	var users []models.User

	database.DB.Find(&users)

	return ctx.JSON(users)
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
