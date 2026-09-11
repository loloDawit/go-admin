package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/models"
)

func GetAllPermissions(ctx *fiber.Ctx) error {
	var permissons []models.Permission

	database.DB.Find(&permissons)

	return ctx.JSON(permissons)
}

func CreatePermission(ctx *fiber.Ctx) error {
	var permissons models.Permission

	if err := ctx.BodyParser(&permissons); err != nil {
		return err
	}

	database.DB.Create(&permissons)
	return ctx.JSON(permissons)
}
