package controllers

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllPermissions(ctx *fiber.Ctx) error {
	var permissons []models.Permission

	database.DB.Find(&permissons)

	return ctx.JSON(permissons)
}

func CreatPermissons(ctx *fiber.Ctx) error {
	var permissons models.Permission

	if err := ctx.BodyParser(&permissons); err != nil {
		return err
	}

	database.DB.Create(&permissons)
	return ctx.JSON(permissons)
}
