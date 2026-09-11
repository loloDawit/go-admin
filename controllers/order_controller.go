package controllers

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
)

func GetAllOrders(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("perPage", "0"))
	return ctx.JSON(models.Paginate(database.DB, &models.Order{}, page, perPage))
}

func GetOrder(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var order models.Order
	if err := database.DB.Preload("OrderItems").First(&order, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "order")
	}
	return ctx.JSON(order)
}

func UpdateOrder(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var req httpx.OrderRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "order")
	}

	if err := database.DB.Model(&order).Updates(map[string]any{
		"email": req.Email,
	}).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(order)
}

func DeleteOrder(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	result := database.DB.Delete(&models.Order{}, id)
	if result.Error != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(result.Error))
	}
	if result.RowsAffected == 0 {
		return httpx.Fail(ctx, errs.NotFound)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func CreateOrder(ctx *fiber.Ctx) error {
	var req httpx.OrderRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	orderItems := make([]models.OrderItem, len(req.OrderItems))
	for i, item := range req.OrderItems {
		orderItems[i] = models.OrderItem{
			ProductTitle: item.ProductTitle,
			Price:        item.Price,
			Quantity:     item.Quantity,
		}
	}

	order := models.Order{
		Email:      req.Email,
		OrderItems: orderItems,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	return ctx.Status(fiber.StatusCreated).JSON(order)
}

func Export(ctx *fiber.Ctx) error {
	filePath := "./csv/orders.csv"
	if err := CreateFile(filePath); err != nil {
		return err
	}
	return ctx.Download(filePath)
}

func CreateFile(filePath string) error {
	file, err := os.Create(filePath)

	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	var orders []models.Order

	database.DB.Preload("OrderItems").Find(&orders)
	writer.Write([]string{
		"ID", "Name", "Email", "Product Title", "Price", "Quantity",
	})

	for _, order := range orders {
		data := []string{
			strconv.Itoa(int(order.Id)), order.FirstName + " " + order.LastName, order.Email, "", "", "",
		}
		if err := writer.Write(data); err != nil {
			return err
		}

		for _, orderItem := range order.OrderItems {
			data = []string{
				"", "", "", orderItem.ProductTitle, strconv.Itoa(int(orderItem.Price)), strconv.Itoa(int(orderItem.Quantity)),
			}
			if err := writer.Write(data); err != nil {
				return err
			}
		}
	}
	return nil
}

type Sales struct {
	Date string `json:"date"`
	Sum  string `json:"sum"`
}

func Chart(ctx *fiber.Ctx) error {
	var sales []Sales

	database.DB.Raw(`
		SELECT DATE_FORMAT (o.created_at, '%Y-%m-%d') AS date, SUM(oi.price * oi.quantity) as sum
		FROM orders o JOIN order_items oi on o.id = oi.order_id
		GROUP BY date;
      `).Scan(&sales)

	return ctx.JSON(sales)
}
