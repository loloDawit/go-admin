// Package httpx maps application errors onto HTTP responses. It is the only
// place that knows both errs and Fiber, keeping the error registry free of
// any web-framework dependency.
package httpx

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/errs"
)

// Body is the single error shape for the whole API. The original code
// returned {"error": ...}, {"msg": ...}, and raw GORM error objects from
// different handlers, so no client could parse failures reliably — and one
// path serialized driver internals straight to the caller.
type Body struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Fail writes err as JSON using its registered status and code. Any error
// outside the registry becomes a generic 500, and its real text is logged
// rather than returned — an unexpected failure must never leak to a client.
//
// Handlers should `return httpx.Fail(ctx, errs.NotFound)` rather than
// building a status and message inline.
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
