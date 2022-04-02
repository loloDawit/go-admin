package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllRoles(ctx *fiber.Ctx) error {
	var roles []models.Role

	database.DB.Find(&roles)

	return ctx.JSON(roles)
}

func GetRole(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	role := models.Role{
		Id: uint(id),
	}

	database.DB.Find(&role)

	return ctx.JSON(role)
}

func UpdateRole(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	role := models.Role{
		Id: uint(id),
	}

	if err := ctx.BodyParser(&role); err != nil {
		return err
	}

	database.DB.Model(&role).Updates(role)

	return ctx.JSON(role)
}

func DeleteRole(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	role := models.Role{
		Id: uint(id),
	}

	database.DB.Delete(&role)

	return ctx.JSON(fiber.Map{
		"msg": "success",
	})
}

func CreateRole(ctx *fiber.Ctx) error {
	var role models.Role

	if err := ctx.BodyParser(&role); err != nil {
		return err
	}

	database.DB.Create(&role)
	return ctx.JSON(role)
}
