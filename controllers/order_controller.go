package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllOrders(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(models.Paginate(database.DB, &models.Order{}, page))
}

func GetOrder(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	order := models.Order{
		Id: uint(id),
	}

	database.DB.Find(&order)

	return ctx.JSON(order)
}

func UpdateOrder(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	order := models.Order{
		Id: uint(id),
	}

	if err := ctx.BodyParser(&order); err != nil {
		return err
	}

	database.DB.Model(&order).Updates(order)

	return ctx.JSON(order)
}

func DeleteOrder(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	order := models.Order{
		Id: uint(id),
	}

	database.DB.Delete(&order)

	return ctx.JSON(fiber.Map{
		"msg": "success",
	})
}

func CreateOrder(ctx *fiber.Ctx) error {
	var order models.Order

	if err := ctx.BodyParser(&order); err != nil {
		return err
	}

	database.DB.Create(&order)
	return ctx.JSON(order)
}
