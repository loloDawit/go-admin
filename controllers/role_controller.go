package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
	"gorm.io/gorm"
)

func GetAllRoles(ctx *fiber.Ctx) error {
	var roles []models.Role

	database.DB.Find(&roles)

	return ctx.JSON(roles)
}

func GetRole(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var role models.Role
	if err := database.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "role")
	}
	return ctx.JSON(role)
}

// The existence check prevents an unknown id from being silently upserted
// as a blank row: GORM's default many2many save creates any Permission that
// doesn't already exist rather than rejecting it.
func loadPermissions(ids []uint) ([]models.Permission, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	unique := make(map[uint]bool, len(ids))
	for _, id := range ids {
		unique[id] = true
	}

	var permissions []models.Permission
	if err := database.DB.Find(&permissions, ids).Error; err != nil {
		return nil, errs.Database.Wrap(err)
	}
	if len(permissions) != len(unique) {
		return nil, errs.NotFound.WithMessage("one or more permissions not found")
	}
	return permissions, nil
}

func UpdateRole(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var req httpx.RoleRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}
	if req.Name == "" {
		return httpx.Fail(ctx, errs.MissingField.WithMessage("name is required"))
	}

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "role")
	}

	permissions, err := loadPermissions(req.Permissions)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	// Name and permissions must land together: a failure between the two
	// statements must not leave the rename committed on its own.
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&role).Updates(map[string]any{
			"name": req.Name,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&role).Association("Permissions").Replace(permissions)
	})
	if err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	if err := database.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "role")
	}
	return ctx.JSON(role)
}

func DeleteRole(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	result := database.DB.Delete(&models.Role{}, id)
	if result.Error != nil {
		if isForeignKeyError(result.Error) {
			return httpx.Fail(ctx, errs.ResourceInUse.Wrap(result.Error))
		}
		return httpx.Fail(ctx, errs.Database.Wrap(result.Error))
	}
	if result.RowsAffected == 0 {
		return httpx.Fail(ctx, errs.NotFound)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func CreateRole(ctx *fiber.Ctx) error {
	var req httpx.RoleRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}
	if req.Name == "" {
		return httpx.Fail(ctx, errs.MissingField.WithMessage("name is required"))
	}

	permissions, err := loadPermissions(req.Permissions)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	role := models.Role{
		Name:        req.Name,
		Permissions: permissions,
	}

	if err := database.DB.Create(&role).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	return ctx.Status(fiber.StatusCreated).JSON(role)
}
