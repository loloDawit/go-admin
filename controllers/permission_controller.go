package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
)

func GetAllPermissions(ctx *fiber.Ctx) error {
	var permissions []models.Permission

	database.DB.Find(&permissions)

	return ctx.JSON(permissions)
}

func CreatePermission(ctx *fiber.Ctx) error {
	var req httpx.PermissionRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}
	if req.Name == "" {
		return httpx.Fail(ctx, errs.MissingField.WithMessage("name is required"))
	}

	permission := models.Permission{Name: req.Name}

	if err := database.DB.Create(&permission).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	return ctx.Status(fiber.StatusCreated).JSON(permission)
}
