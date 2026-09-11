package controllers

import (
	"github.com/loloDawit/go-admin/middlewares"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/models"
)

func GetAllUsers(ctx *fiber.Ctx) error {
	if err := middlewares.IsAuthorized(ctx, "users"); err != nil {
		return err
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(models.Paginate(database.DB, &models.User{}, page))
}

func GetUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsAuthorized(ctx, "users"); err != nil {
		return err
	}
	id, _ := strconv.Atoi(ctx.Params("id"))

	user := models.User{
		Id: id,
	}

	database.DB.Preload("Role").Find(&user)

	return ctx.JSON(user)
}

func UpdateUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsAuthorized(ctx, "users"); err != nil {
		return err
	}
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
	if err := middlewares.IsAuthorized(ctx, "users"); err != nil {
		return err
	}
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
	if err := middlewares.IsAuthorized(ctx, "users"); err != nil {
		return err
	}
	var user models.User

	if err := ctx.BodyParser(&user); err != nil {
		return err
	}

	// The initial password is still hardcoded here; Task 6 replaces this with
	// an admin-supplied value once registration becomes invite-only. Until
	// then this at least fails loudly rather than silently storing a digest
	// of a string nobody knows.
	if err := user.SetPassword(hasher, "124"); err != nil {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{"error": err.Error()})
	}

	if err := database.DB.Create(&user).Error; err != nil {
		ctx.Status(400)
		return ctx.JSON(fiber.Map{"error": "could not create the user"})
	}
	return ctx.JSON(user)
}
