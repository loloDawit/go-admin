package models

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	defaultPerPage = 25
	maxPerPage     = 100
	maxPage        = 1_000_000 // bounds (page-1)*perPage against overflow from an oversized ?page=
)

func lastPage(total int64, perPage int) int {
	if total <= 0 {
		return 1
	}
	pages := int((total + int64(perPage) - 1) / int64(perPage))
	if pages < 1 {
		return 1
	}
	return pages
}

func normalizePerPage(perPage int) int {
	switch {
	case perPage <= 0:
		return defaultPerPage
	case perPage > maxPerPage:
		return maxPerPage
	default:
		return perPage
	}
}

func normalizePage(page int) int {
	switch {
	case page < 1:
		return 1
	case page > maxPage:
		return maxPage
	default:
		return page
	}
}

func Paginate(db *gorm.DB, entity Entity, page, perPage int) fiber.Map {
	page = normalizePage(page)
	perPage = normalizePerPage(perPage)
	offset := (page - 1) * perPage

	data := entity.Take(db, perPage, offset)
	total := entity.Count(db)

	return fiber.Map{
		"data": data,
		"meta": fiber.Map{
			"total":     total,
			"page":      page,
			"perPage":   perPage,
			"last_page": lastPage(total, perPage),
		},
	}
}
