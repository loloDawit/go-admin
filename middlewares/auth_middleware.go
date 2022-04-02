package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/utils"
)

func IsAuthenticated(ctx *fiber.Ctx) error {
	cookie := ctx.Cookies("jwt")

	if _, err := utils.ParseJWT(cookie); err != nil {
		ctx.Status(fiber.StatusUnauthorized)
		return ctx.JSON(fiber.Map{
			"msg": "unauthorized.",
		})
	}

	return ctx.Next()
}
