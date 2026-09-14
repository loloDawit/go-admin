package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
)

type Service struct {
	repo     Repository
	store    Store
	maxBytes int64
}

func NewService(repo Repository, store Store, maxBytes int64) *Service {
	return &Service{repo: repo, store: store, maxBytes: maxBytes}
}

// Upload validates f against IMAGE_MAX_BYTES and the sniffed content type,
// then puts the object before inserting its row: on insert failure the
// object is deleted so nothing points at it, since an orphaned object is
// waste but an orphaned row is a broken image on a page someone is looking
// at.
func (s *Service) Upload(ctx context.Context, productID int64, f File) (Image, error) {
	if f.Size > s.maxBytes {
		return Image{}, ErrImageTooLarge
	}

	// Content-Length is a claim, not a fact: MaxBytesReader still bounds the
	// actual read regardless of what f.Size said.
	body := http.MaxBytesReader(nil, f.Body, s.maxBytes)
	defer body.Close()

	peek := make([]byte, sniffLen)
	n, err := io.ReadFull(body, peek)
	switch {
	case err == nil, errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, io.EOF):
		peek = peek[:n]
	default:
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return Image{}, ErrImageTooLarge
		}
		return Image{}, errs.Wrap(errs.OpUploadImage, err)
	}

	sniffed := http.DetectContentType(peek)
	ext, ok := allowedImageTypes[sniffed]
	if !ok || sniffed != f.ContentType {
		return Image{}, ErrUnsupportedImageType
	}

	key := objectKey(productID, ext)
	full := io.MultiReader(bytes.NewReader(peek), body)
	if err := s.store.Put(ctx, key, full, f.Size, sniffed); err != nil {
		return Image{}, errs.Wrap(errs.OpUploadImage, err)
	}

	created, err := s.repo.Create(ctx, Image{ProductID: productID, ObjectKey: key})
	if err != nil {
		if delErr := s.store.Delete(ctx, key); delErr != nil {
			return Image{}, errs.Wrap(errs.OpUploadImage, fmt.Errorf("row insert failed (%v) and object cleanup also failed: %w", err, delErr))
		}
		return Image{}, errs.Wrap(errs.OpUploadImage, err)
	}
	return created, nil
}

// Delete removes the object before the row: a row surviving with no object
// behind it is a broken image; an object surviving with no row naming it is
// merely unreachable waste.
func (s *Service) Delete(ctx context.Context, productID, imageID int64) error {
	img, err := s.repo.Get(ctx, productID, imageID)
	if err != nil {
		if errors.Is(err, ErrImageNotFound) {
			return err
		}
		return errs.Wrap(errs.OpDeleteImage, err)
	}

	if err := s.store.Delete(ctx, img.ObjectKey); err != nil {
		return errs.Wrap(errs.OpDeleteImage, err)
	}
	if err := s.repo.Delete(ctx, productID, imageID); err != nil {
		if errors.Is(err, ErrImageNotFound) {
			return err
		}
		return errs.Wrap(errs.OpDeleteImage, err)
	}
	return nil
}

// URLsFor answers a product's images with a URL presigned fresh for this
// call, never a stored one: a stored URL outlives its TTL and starts failing
// for reasons nobody can see.
func (s *Service) URLsFor(ctx context.Context, productID int64) ([]ImageResponse, error) {
	imgs, err := s.repo.ListByProduct(ctx, productID)
	if err != nil {
		return nil, errs.Wrap(errs.OpListImages, err)
	}

	out := make([]ImageResponse, 0, len(imgs))
	for _, img := range imgs {
		url, err := s.store.Presign(ctx, img.ObjectKey)
		if err != nil {
			return nil, errs.Wrap(errs.OpListImages, err)
		}
		out = append(out, newImageResponse(img, url))
	}
	return out, nil
}

// PresignURL lets a caller holding an Image (from Upload, say) build the one
// response field an ObjectKey is ever allowed to produce.
func (s *Service) PresignURL(ctx context.Context, objectKey string) (string, error) {
	url, err := s.store.Presign(ctx, objectKey)
	if err != nil {
		return "", errs.Wrap(errs.OpUploadImage, err)
	}
	return url, nil
}

// objectKey is generated entirely server-side: productID and a fresh UUID,
// with an extension taken only from the sniffed content type. Nothing the
// client sent reaches this path.
func objectKey(productID int64, ext string) string {
	return fmt.Sprintf("products/%d/%s.%s", productID, uuid.NewString(), ext)
}
