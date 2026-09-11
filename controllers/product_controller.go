package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/models"
)

func GetAllProducts(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("perPage", "0"))
	return ctx.JSON(models.Paginate(database.DB, &models.Product{}, page, perPage))
}

func GetProduct(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.Product{
		Id: uint(id),
	}

	database.DB.Find(&user)

	return ctx.JSON(user)
}

func UpdateProduct(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.Product{
		Id: uint(id),
	}

	if err := ctx.BodyParser(&user); err != nil {
		return err
	}

	database.DB.Model(&user).Updates(user)

	return ctx.JSON(user)
}

func DeleteProduct(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.Product{
		Id: uint(id),
	}

	database.DB.Delete(&user)

	return ctx.JSON(fiber.Map{
		"msg": "success",
	})
}

func CreateProduct(ctx *fiber.Ctx) error {
	var user models.Product

	if err := ctx.BodyParser(&user); err != nil {
		return err
	}

	database.DB.Create(&user)
	return ctx.JSON(user)
}
