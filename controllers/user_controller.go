package controllers

import (
	"github.com/loloDawit/go-admin/middlewares"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
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

	var req httpx.CreateUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	user := models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		RoleId:    req.RoleId,
	}

	if err := user.Validate(); err != nil {
		return httpx.Fail(ctx, err)
	}
	// Without this check an unknown roleId fails Create on the FK constraint,
	// which the generic branch below would misreport as EmailTaken.
	if err := database.DB.First(&models.Role{}, req.RoleId).Error; err != nil {
		return notFoundOrDBError(ctx, err, "role")
	}
	if err := user.SetPassword(hasher, req.Password); err != nil {
		return httpx.Fail(ctx, err)
	}

	if err := database.DB.Create(&user).Error; err != nil {
		if isDuplicateKeyError(err) {
			return httpx.Fail(ctx, errs.EmailTaken.Wrap(err))
		}
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.Status(fiber.StatusCreated).JSON(user)
}
