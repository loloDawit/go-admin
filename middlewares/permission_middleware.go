package middlewares

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
	"github.com/loloDawit/go-admin/utils"
)

// RequirePermission enforces the view_/edit_ convention: safe methods accept
// either, mutating methods require edit_. Attached at every resource route so
// authorization is the default rather than opt-in.
func RequirePermission(resource string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		issuer, err := utils.ParseJWT(ctx.Cookies("jwt"))
		if err != nil {
			return httpx.Fail(ctx, errs.Unauthenticated)
		}

		userId, err := strconv.Atoi(issuer)
		if err != nil {
			return httpx.Fail(ctx, errs.SessionInvalid)
		}

		var user models.User
		if err := database.DB.First(&user, userId).Error; err != nil {
			return httpx.Fail(ctx, errs.SessionInvalid.Wrap(err))
		}

		if user.RoleId == 0 {
			return httpx.Fail(ctx, errs.NoRoleAssigned)
		}

		var role models.Role
		if err := database.DB.Preload("Permissions").First(&role, user.RoleId).Error; err != nil {
			return httpx.Fail(ctx, errs.Database.Wrap(err))
		}

		required := "edit_" + resource
		alsoAccepted := ""
		if isSafeMethod(ctx.Method()) {
			alsoAccepted = "view_" + resource
		}

		for _, p := range role.Permissions {
			if p.Name == required || (alsoAccepted != "" && p.Name == alsoAccepted) {
				ctx.Locals("userId", userId)
				return ctx.Next()
			}
		}

		return httpx.Fail(ctx, errs.Forbidden)
	}
}

func isSafeMethod(method string) bool {
	return method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions
}
