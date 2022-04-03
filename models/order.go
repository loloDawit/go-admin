package models

type Order struct {
	Id        uint   `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastLame"`
	Email     string `json:"email"`
	UpdatedAt string `json:"updatedAt"`
	CreatedAt string `json:"createdAt"`
	OrderItem []OrderItem `json:"orderItems" gorm:"foreignKey:OrderId"`
}

type OrderItem struct {
	Id           uint    `json:"id"`
	OrderId      uint    `json:"orderId"`
	ProductTitle string  `json:"productTitle"`
	Price        float32 `json:"price"`
	Quantity     uint    `json:"updatedAt"`
}
