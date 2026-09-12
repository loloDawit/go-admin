package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/utils"
)

func IsAuthenticated(ctx *fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")

	if _, err := utils.ParseJWT(cookie); err != nil {
		return httpx.Fail(ctx, errs.Unauthenticated)
	}

	return ctx.Next()
}
