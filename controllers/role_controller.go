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
	var roleDTO fiber.Map

	if err := ctx.BodyParser(&roleDTO); err != nil {
		return err
	}

	list := roleDTO["permissions"].([]interface{})

	permissions := make([]models.Permission, len(list))

	for i, permissionId := range list {
		id, _ := strconv.Atoi(permissionId.(string))

		permissions[i] = models.Permission{
			Id: uint(id),
		}
	}

	// remove old permission and attach a new one
	var result interface{}

	database.DB.Table("role_permissions").Where("role_id", id).Delete(result)

	role := models.Role{
		Id:          uint(id),
		Name:        roleDTO["name"].(string),
		Permissions: permissions,
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
	var roleDTO fiber.Map

	if err := ctx.BodyParser(&roleDTO); err != nil {
		return err
	}

	list := roleDTO["permissions"].([]interface{})

	permissions := make([]models.Permission, len(list))

	for i, permissionId := range list {
		id, _ := strconv.Atoi(permissionId.(string))

		permissions[i] = models.Permission{
			Id: uint(id),
		}
	}

	role := models.Role{
		Name:        roleDTO["name"].(string),
		Permissions: permissions,
	}

	database.DB.Create(&role)
	return ctx.JSON(role)
}
