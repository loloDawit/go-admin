package controllers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"gorm.io/gorm"
)

func pathId(ctx *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return 0, errs.InvalidID
	}
	return id, nil
}

// Callers must use First, not Find: Find returns a zero-valued struct and no
// error for a missing row.
func notFoundOrDBError(ctx *fiber.Ctx, err error, resource string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return httpx.Fail(ctx, errs.NotFound.WithMessage("%s not found", resource))
	}
	return httpx.Fail(ctx, errs.Database.Wrap(err))
}
