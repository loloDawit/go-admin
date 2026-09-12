package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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

		if err := saveUpload(header, dest); err != nil {
			return httpx.Fail(ctx, err)
		}

		return ctx.JSON(fiber.Map{
			"url": strings.TrimRight(cfg.PublicBaseURL, "/") + config.UploadsPath + "/" + name,
		})
	}
}

func sniffImageType(header *multipart.FileHeader) (string, error) {
	f, err := header.Open()
	if err != nil {
		return "", errs.UploadFailed.Wrap(err)
	}
	defer f.Close()

	// ReadFull: a single Read may legally return fewer than 512 bytes
	// without EOF, and a short read must not be mistaken for the file's
	// real content.
	buf := make([]byte, 512)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", errs.UploadFailed.Wrap(err)
	}

	sniffed := strings.SplitN(http.DetectContentType(buf[:n]), ";", 2)[0]
	ext, ok := allowedImageTypes[sniffed]
	if !ok {
		return "", errs.UploadUnsupportedType
	}
	return ext, nil
}

// saveUpload writes header's content to dest. O_EXCL, not ctx.SaveFile's
// rename-or-truncate, so a stored-name collision fails instead of silently
// overwriting the existing file.
func saveUpload(header *multipart.FileHeader, dest string) error {
	src, err := header.Open()
	if err != nil {
		return errs.UploadFailed.Wrap(err)
	}
	defer src.Close()

	return copyToFile(src, dest)
}

// copyToFile writes src to a newly created dest. A copy failure removes the
// partial file OpenFile created, so a broken upload never leaves stored
// bytes behind; removal is best-effort and never replaces the original error.
func copyToFile(src io.Reader, dest string) error {
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errs.UploadFailed
		}
		return errs.UploadFailed.Wrap(err)
	}

	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		os.Remove(dest)
		return errs.UploadFailed.Wrap(err)
	}
	if err := out.Close(); err != nil {
		os.Remove(dest)
		return errs.UploadFailed.Wrap(err)
	}
	return nil
}

func randomName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ext, nil
}
