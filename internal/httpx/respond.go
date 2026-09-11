// Package httpx maps application errors onto HTTP responses.
package httpx

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/errs"
)

// Body is the one error shape the API returns.
type Body struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Fail writes err using its registered status and code. Unregistered errors
// become a generic 500 and are logged rather than returned.
func Fail(ctx *fiber.Ctx, err error) error {
	appErr := errs.From(err)

	if cause := appErr.Cause(); cause != nil {
		log.Printf("error: %s %s: %s: %v", ctx.Method(), ctx.Path(), appErr.Code, cause)
	} else if appErr.Status >= 500 {
		log.Printf("error: %s %s: %s", ctx.Method(), ctx.Path(), appErr.Code)
	}

	return ctx.Status(appErr.Status).JSON(Body{
		Code:    appErr.Code,
		Message: appErr.Message,
	})
}
