package controllers

import (
	"encoding/csv"
	"io"
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
	order.Compute()
	return ctx.JSON(order)
}

func UpdateOrder(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var req httpx.UpdateOrderRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
	}

	var order models.Order
	if err := database.DB.Preload("OrderItems").First(&order, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "order")
	}

	// req.Email == "" means the field was omitted, not that a client asked
	// to clear it: the map form would otherwise write "" whenever a request
	// leaves email out.
	if req.Email != "" {
		if err := database.DB.Model(&order).Updates(map[string]any{
			"email": req.Email,
		}).Error; err != nil {
			return httpx.Fail(ctx, errs.Database.Wrap(err))
		}
	}

	order.Compute()
	return ctx.JSON(order)
}

func DeleteOrder(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	result := database.DB.Delete(&models.Order{}, id)
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

func CreateOrder(ctx *fiber.Ctx) error {
	var req httpx.OrderRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody.Wrap(err))
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
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		OrderItems: orderItems,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	order.Compute()
	return ctx.Status(fiber.StatusCreated).JSON(order)
}

func Export(ctx *fiber.Ctx) error {
	// Per-request: a shared path corrupts concurrent exports.
	file, err := os.CreateTemp("", "orders-*.csv")
	if err != nil {
		return httpx.Fail(ctx, errs.ExportFailed)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if err := writeOrdersCSV(file); err != nil {
		return httpx.Fail(ctx, errs.ExportFailed)
	}

	ctx.Set("Content-Disposition", `attachment; filename="orders.csv"`)
	// SendFile's own error embeds the temp path; a 5xx must not leak a
	// filesystem path to the client.
	if err := ctx.SendFile(file.Name()); err != nil {
		return httpx.Fail(ctx, errs.ExportFailed)
	}
	return nil
}

func writeOrdersCSV(w io.Writer) error {
	writer := csv.NewWriter(w)

	var orders []models.Order
	if err := database.DB.Preload("OrderItems").Find(&orders).Error; err != nil {
		return err
	}

	if err := writer.Write([]string{"ID", "Name", "Email", "Product Title", "Price", "Quantity"}); err != nil {
		return err
	}

	for _, order := range orders {
		order.Compute()
		if err := writer.Write([]string{
			strconv.FormatUint(uint64(order.Id), 10),
			order.Name,
			order.Email, "", "", "",
		}); err != nil {
			return err
		}
		for _, item := range order.OrderItems {
			if err := writer.Write([]string{
				"", "", "", item.ProductTitle,
				strconv.FormatFloat(item.Price, 'f', 2, 64),
				strconv.FormatUint(uint64(item.Quantity), 10),
			}); err != nil {
				return err
			}
		}
	}

	// Explicit Flush before Error: a deferred Flush runs after the return
	// value is already evaluated, silently dropping a disk-full failure.
	writer.Flush()
	return writer.Error()
}

type Sales struct {
	Date string  `json:"date"`
	Sum  float64 `json:"sum"`
}

func Chart(ctx *fiber.Ctx) error {
	var sales []Sales

	// Requires created_at to be a real DATETIME. DATE_FORMAT (not DATE) in
	// the projection: with parseTime=true the driver scans a DATE column
	// into time.Time, not the string Sales.Date expects. GROUP BY repeats
	// the same expression: only_full_group_by rejects DATE_FORMAT in the
	// SELECT list when the GROUP BY is the plain DATE() form.
	err := database.DB.Raw(`
		SELECT DATE_FORMAT(o.created_at, '%Y-%m-%d') AS date,
		       SUM(oi.price * oi.quantity) AS sum
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		GROUP BY DATE_FORMAT(o.created_at, '%Y-%m-%d')
		ORDER BY date
	`).Scan(&sales).Error
	if err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(sales)
}
