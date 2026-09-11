package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
)

func GetAllProducts(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("perPage", "0"))
	return ctx.JSON(models.Paginate(database.DB, &models.Product{}, page, perPage))
}

func GetProduct(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "product")
	}
	return ctx.JSON(product)
}

func CreateProduct(ctx *fiber.Ctx) error {
	var req httpx.ProductRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}
	if req.Title == "" {
		return httpx.Fail(ctx, errs.TitleRequired)
	}
	if req.Price < 0 {
		return httpx.Fail(ctx, errs.PriceInvalid)
	}

	product := models.Product{
		Title:       req.Title,
		Description: req.Description,
		Image:       req.Image,
		Price:       req.Price,
	}

	if err := database.DB.Create(&product).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	return ctx.Status(fiber.StatusCreated).JSON(product)
}

func UpdateProduct(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var req httpx.ProductRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "product")
	}

	// Must be a map: the struct form skips zero values. Id comes from the
	// path only.
	if err := database.DB.Model(&product).Updates(map[string]any{
		"title":       req.Title,
		"description": req.Description,
		"image":       req.Image,
		"price":       req.Price,
	}).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(product)
}

func DeleteProduct(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	result := database.DB.Delete(&models.Product{}, id)
	if result.Error != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(result.Error))
	}
	if result.RowsAffected == 0 {
		return httpx.Fail(ctx, errs.NotFound)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
