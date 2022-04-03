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
