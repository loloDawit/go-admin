package models

import (
	"math"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Paginate(db *gorm.DB, entity Entity, page int) fiber.Map {
	limit := 5
	offset := (page - 1) * limit

	data := entity.Take(db, limit, offset)
	total := entity.Count(db)

	lastPage := math.Ceil(float64(int(total) / limit))
	return fiber.Map{
		"data": data,
		"meta": fiber.Map{
			"last_page": lastPage,
			"total":     total,
			"page":      page,
		},
	}
}
