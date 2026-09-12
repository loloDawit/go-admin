package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type Order struct {
	Id        uint   `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`

	Name  string  `json:"name" gorm:"-"`
	Total float64 `json:"total" gorm:"-"`

	// Must be time.Time: GORM only auto-populates these when they are.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	OrderItems []OrderItem `json:"orderItems" gorm:"foreignKey:OrderId"`
}

type OrderItem struct {
	Id           uint    `json:"id"`
	OrderId      uint    `json:"orderId"`
	ProductTitle string  `json:"productTitle"`
	Price        float64 `json:"price"`
	Quantity     uint    `json:"quantity"`
}

// Compute fills the derived fields. Call it on every path that returns an
// Order, since Name and Total are not persisted.
func (order *Order) Compute() {
	order.Name = strings.TrimSpace(order.FirstName + " " + order.LastName)

	var total float64
	for _, item := range order.OrderItems {
		total += item.Price * float64(item.Quantity)
	}
	order.Total = total
}

func (order *Order) Count(db *gorm.DB) int64 {
	var total int64
	db.Model(&Order{}).Count(&total)

	return total
}

func (order *Order) Take(db *gorm.DB, limit int, offset int) interface{} {
	var orders []Order

	db.Preload("OrderItems").Offset(offset).Limit(limit).Find(&orders)

	for i := range orders {
		orders[i].Compute()
	}
	return orders
}
