package controllers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"gorm.io/gorm"
)

// pathId parses and validates the :id path parameter.
//
// The original used `id, _ := strconv.Atoi(ctx.Params("id"))` in every
// handler, so /product/abc silently became product 0 (ASSESSMENT 4q).
func pathId(ctx *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return 0, errs.InvalidID
	}
	return id, nil
}

// notFoundOrDBError maps a GORM lookup failure to the right response.
//
// Find on a missing row returns a zero-valued struct and NO error, which is
// why the original handlers answered 200 with empty objects; handlers must
// use First and route its error through here (ASSESSMENT 4o).
func notFoundOrDBError(ctx *fiber.Ctx, err error, resource string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return httpx.Fail(ctx, errs.NotFound.WithMessage("%s not found", resource))
	}
	// Wrap, so the driver's text reaches the log but never the client.
	return httpx.Fail(ctx, errs.Database.Wrap(err))
}
