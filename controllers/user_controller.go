package controllers

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/models"
)

func GetAllUsers(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit := 5
	offset := (page - 1) * limit
	var total int64

	var users []models.User

	database.DB.Preload("Role").Offset(offset).Limit(limit).Find(&users)
	database.DB.Model(&models.User{}).Count(&total)

	lastPage := math.Ceil(float64(int(total) / limit))
	return ctx.JSON(fiber.Map{
		"data": users,
		"meta": fiber.Map{
			"last_page": lastPage,
			"total":     total,
			"page":      page,
		},
	})
}

func GetUser(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	database.DB.Preload("Role").Find(&user)

	return ctx.JSON(user)
}

func UpdateUser(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	if err := ctx.BodyParser(&user); err != nil {
		return err
	}

	database.DB.Model(&user).Updates(user)

	return ctx.JSON(user)
}

func DeleteUser(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	database.DB.Delete(&user)

	return ctx.JSON(fiber.Map{
		"msg": "success",
	})
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
