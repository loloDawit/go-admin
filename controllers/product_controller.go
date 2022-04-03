package controllers

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllProducts(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit := 5
	offset := (page - 1) * limit
	var total int64

	var products []models.Product

	database.DB.Offset(offset).Limit(limit).Find(&products)
	database.DB.Model(&models.Product{}).Count(&total)

	lastPage := math.Ceil(float64(int(total) / limit))
	return ctx.JSON(fiber.Map{
		"data": products,
		"meta": fiber.Map{
			"last_page": lastPage,
			"total":     total,
			"page":      page,
		},
	})
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
