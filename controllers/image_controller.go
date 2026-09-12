package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/httpx"
)

// Checked against sniffed content, never the client's filename or
// Content-Type header — both are attacker-controlled.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Upload stores one image under a generated name and returns its public URL.
func Upload(cfg *config.Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		form, err := ctx.MultipartForm()
		if err != nil {
			return httpx.Fail(ctx, errs.UploadMalformed)
		}

		files := form.File["image"]
		if len(files) != 1 {
			return httpx.Fail(ctx, errs.UploadMalformed)
		}
		header := files[0]

		if header.Size > cfg.MaxUploadBytes {
			return httpx.Fail(ctx, errs.UploadTooLarge.WithMessage(
				"the file exceeds the %d byte limit", cfg.MaxUploadBytes))
		}

		ext, err := sniffImageType(header)
		if err != nil {
			return httpx.Fail(ctx, err)
		}

		// Random: no client-supplied bytes (filename or extension) reach
		// the filesystem.
		name, err := randomName(ext)
		if err != nil {
			return httpx.Fail(ctx, errs.UploadFailed.Wrap(err))
		}

		dest := filepath.Join(cfg.UploadDir, name)

		// Defence in depth even though name is generated, not derived from
		// client input.
		absDir, err := filepath.Abs(cfg.UploadDir)
		if err != nil {
			return httpx.Fail(ctx, errs.UploadFailed.Wrap(err))
		}
		absDest, err := filepath.Abs(dest)
		if err != nil {
			return httpx.Fail(ctx, errs.UploadFailed.Wrap(err))
		}
		if absDest != filepath.Join(absDir, name) || strings.ContainsAny(name, `/\`) {
			return httpx.Fail(ctx, errs.UploadFailed)
		}

		if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
			return httpx.Fail(ctx, errs.UploadFailed.Wrap(err))
		}

		if err := ctx.SaveFile(header, dest); err != nil {
			return httpx.Fail(ctx, errs.UploadFailed.Wrap(err))
		}

		return ctx.JSON(fiber.Map{
			"url": strings.TrimRight(cfg.PublicBaseURL, "/") + "/api/v1/uploads/" + name,
		})
	}
}

// sniffImageType returns the extension for header's content.
func sniffImageType(header *multipart.FileHeader) (string, error) {
	f, err := header.Open()
	if err != nil {
		return "", errs.UploadFailed.Wrap(err)
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", errs.UploadFailed.Wrap(err)
	}

	sniffed := strings.SplitN(http.DetectContentType(buf[:n]), ";", 2)[0]
	ext, ok := allowedImageTypes[sniffed]
	if !ok {
		return "", errs.UploadUnsupportedType
	}
	return ext, nil
}

func randomName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ext, nil
}
