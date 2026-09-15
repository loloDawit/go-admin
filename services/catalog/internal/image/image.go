// Package image handles product image upload, storage, and deletion. The
// object key it generates never carries anything the client sent; product
// responses carry a presigned URL, never the key itself.
package image

import (
	"io"
	"time"

	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
)

var (
	ErrUnsupportedImageType = errs.ErrUnsupportedImageType
	ErrImageTooLarge        = errs.ErrImageTooLarge
	ErrImageNotFound        = errs.ErrImageNotFound
)

// sniffLen is how many leading bytes http.DetectContentType inspects.
const sniffLen = 512

// allowedImageTypes maps a sniffed content type to the extension its object
// key carries; a sniffed type absent here, or one that disagrees with what
// the client declared, is a rejection.
var allowedImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// Image is a stored product image. ObjectKey is what the object store and
// database key off of; it must never reach an HTTP response.
type Image struct {
	ID        int64
	ProductID int64
	ObjectKey string
	Alt       string
	Position  int
	CreatedAt time.Time
}

// File is Upload's input. Size and ContentType are the client's claims, not
// facts: Upload verifies both against the bytes themselves before trusting
// either. There is deliberately no filename field: the object key is
// generated server-side and nothing the client sent may reach a path.
type File struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
}
