package controllers

import (
	"encoding/csv"
	"os"
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
