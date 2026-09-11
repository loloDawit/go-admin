package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
)

func GetAllUsers(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("perPage", "0"))
	return ctx.JSON(models.Paginate(database.DB, &models.User{}, page, perPage))
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

	var req httpx.UpdateUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	updates := map[string]any{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"email":      req.Email,
	}

	if req.RoleId != 0 {
		var targetRole models.Role
		if err := database.DB.Preload("Permissions").First(&targetRole, req.RoleId).Error; err != nil {
			return notFoundOrDBError(ctx, err, "role")
		}
		if err := ensureCanAssignRole(ctx, targetRole); err != nil {
			return httpx.Fail(ctx, err)
		}
		updates["role_id"] = req.RoleId
	}

	if err := database.DB.Model(&models.User{Id: id}).Updates(updates).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "user")
	}
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

	var targetRole models.Role
	// Without this check an unknown roleId fails Create on the FK constraint,
	// which the generic branch below would misreport as EmailTaken.
	if err := database.DB.Preload("Permissions").First(&targetRole, req.RoleId).Error; err != nil {
		return notFoundOrDBError(ctx, err, "role")
	}
	if err := ensureCanAssignRole(ctx, targetRole); err != nil {
		return httpx.Fail(ctx, err)
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
