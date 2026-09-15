package image_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/services/catalog/internal/image"
)

// pngHeader is enough of a real PNG signature for http.DetectContentType to
// recognize; the remaining bytes are filler, not a decodable image.
var pngHeader = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func validPNGBytes(total int) []byte {
	b := make([]byte, total)
	copy(b, pngHeader)
	return b
}

// fakeStore is an in-memory double for Store; each test builds its own so
// none depends on another test's state or a real object store.
type fakeStore struct {
	putErr    error
	deleteErr error
	presignN  int
	deleted   []string
	putCalled bool
	callOrder *[]string
}

func newFakeStore() *fakeStore {
	order := []string{}
	return &fakeStore{callOrder: &order}
}

func (s *fakeStore) Put(_ context.Context, _ string, r io.Reader, _ int64, _ string) error {
	s.putCalled = true
	*s.callOrder = append(*s.callOrder, "put")
	if s.putErr != nil {
		return s.putErr
	}
	// Drain fully, mirroring a real store reading the whole reader.
	_, err := io.Copy(io.Discard, r)
	return err
}

func (s *fakeStore) Presign(_ context.Context, key string) (string, error) {
	s.presignN++
	return "https://minio.example/" + key + "?sig=" + strings.Repeat("a", s.presignN), nil
}

func (s *fakeStore) Delete(_ context.Context, key string) error {
	*s.callOrder = append(*s.callOrder, "delete")
	s.deleted = append(s.deleted, key)
	return s.deleteErr
}

// fakeRepository is an in-memory double for Repository.
type fakeRepository struct {
	nextID    int64
	byID      map[int64]image.Image
	createErr error
	callOrder *[]string
}

func newFakeRepository(order *[]string) *fakeRepository {
	return &fakeRepository{byID: make(map[int64]image.Image), callOrder: order}
}

func (r *fakeRepository) Create(_ context.Context, img image.Image) (image.Image, error) {
	if r.callOrder != nil {
		*r.callOrder = append(*r.callOrder, "insert")
	}
	if r.createErr != nil {
		return image.Image{}, r.createErr
	}
	r.nextID++
	img.ID = r.nextID
	r.byID[img.ID] = img
	return img, nil
}

func (r *fakeRepository) Get(_ context.Context, productID, imageID int64) (image.Image, error) {
	img, ok := r.byID[imageID]
	if !ok || img.ProductID != productID {
		return image.Image{}, image.ErrImageNotFound
	}
	return img, nil
}

func (r *fakeRepository) Delete(_ context.Context, productID, imageID int64) error {
	if r.callOrder != nil {
		*r.callOrder = append(*r.callOrder, "row-delete")
	}
	img, ok := r.byID[imageID]
	if !ok || img.ProductID != productID {
		return image.ErrImageNotFound
	}
	delete(r.byID, imageID)
	return nil
}

func (r *fakeRepository) ListByProduct(_ context.Context, productID int64) ([]image.Image, error) {
	var out []image.Image
	for _, img := range r.byID {
		if img.ProductID == productID {
			out = append(out, img)
		}
	}
	return out, nil
}

func newTestService(store *fakeStore, repo *fakeRepository, maxBytes int64) *image.Service {
	return image.NewService(repo, store, maxBytes)
}

// The declared type is attacker-supplied; the sniffed type is evidence. A
// multipart part that declares image/png over bytes that are actually a
// shell script must be refused, not trusted.
func TestUploadRejectsADeclaredTypeThatLies(t *testing.T) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	svc := newTestService(store, repo, 1<<20)

	script := []byte("#!/bin/sh\necho pwned\n")
	_, err := svc.Upload(t.Context(), 1, image.File{
		Body:        io.NopCloser(bytes.NewReader(script)),
		Size:        int64(len(script)),
		ContentType: "image/png",
	})

	if !errors.Is(err, image.ErrUnsupportedImageType) {
		t.Fatalf("want ErrUnsupportedImageType, got %v", err)
	}
	if store.putCalled {
		t.Fatal("Put must not be called once the sniff rejects the upload")
	}
}

// It must refuse without reading the whole body into memory:
// http.MaxBytesReader bounds the read regardless of what Size claimed.
func TestUploadRefusesABodyOverTheCap(t *testing.T) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	const maxBytes = 10
	svc := newTestService(store, repo, maxBytes)

	// Size understates the real body, exactly as an attacker-controlled
	// Content-Length would: the cap must still bind on the actual bytes.
	oversized := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 1000)
	_, err := svc.Upload(t.Context(), 1, image.File{
		Body:        io.NopCloser(bytes.NewReader(oversized)),
		Size:        5,
		ContentType: "image/png",
	})

	if !errors.Is(err, image.ErrImageTooLarge) {
		t.Fatalf("want ErrImageTooLarge, got %v", err)
	}
	if store.putCalled {
		t.Fatal("Put must not be called once the cap rejects the upload")
	}
}

// The object key is products/{id}/{uuid}.{ext}, generated here; File carries
// no filename field at all, so nothing the client sent can reach a path.
func TestObjectKeyIgnoresTheClientFilename(t *testing.T) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	svc := newTestService(store, repo, 1<<20)

	body := validPNGBytes(64)
	created, err := svc.Upload(t.Context(), 42, image.File{
		Body:        io.NopCloser(bytes.NewReader(body)),
		Size:        int64(len(body)),
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if strings.Contains(created.ObjectKey, "..") || strings.Contains(created.ObjectKey, "etc") {
		t.Fatalf("object key must never carry client-supplied path segments, got %q", created.ObjectKey)
	}
	if !strings.HasPrefix(created.ObjectKey, "products/42/") || !strings.HasSuffix(created.ObjectKey, ".png") {
		t.Fatalf("object key: want products/42/{uuid}.png shape, got %q", created.ObjectKey)
	}
}

// Two systems, no shared transaction: if the row insert fails after the
// object was already stored, the object must be deleted rather than left
// orphaned and unreferenced-but-billed-for.
func TestAFailedRowInsertDeletesTheStoredObject(t *testing.T) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	repo.createErr = errors.New("insert failed: connection reset")
	svc := newTestService(store, repo, 1<<20)

	body := validPNGBytes(64)
	_, err := svc.Upload(t.Context(), 1, image.File{
		Body:        io.NopCloser(bytes.NewReader(body)),
		Size:        int64(len(body)),
		ContentType: "image/png",
	})

	if err == nil {
		t.Fatal("want an error when the row insert fails")
	}
	if len(store.deleted) != 1 {
		t.Fatalf("want exactly one Store.Delete call cleaning up the orphaned object, got %d", len(store.deleted))
	}
}

// Row first would leave an unreachable object with nothing naming it, then a
// second failure trying to delete it; deleting the object first means a
// failure between the two steps leaves only that same harmless orphan.
func TestDeleteRemovesTheObjectBeforeTheRow(t *testing.T) {
	order := []string{}
	store := newFakeStore()
	store.callOrder = &order
	repo := newFakeRepository(&order)
	svc := newTestService(store, repo, 1<<20)

	created, err := repo.Create(t.Context(), image.Image{ProductID: 1, ObjectKey: "products/1/abc.png"})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	order = order[:0] // Delete's ordering is what this test proves; the seed insert above is not part of it.

	if err := svc.Delete(t.Context(), 1, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if len(order) != 2 || order[0] != "delete" || order[1] != "row-delete" {
		t.Fatalf("want [delete row-delete] order, got %v", order)
	}
}

func TestProductDetailCarriesPresignedURLs(t *testing.T) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	svc := newTestService(store, repo, 1<<20)

	if _, err := repo.Create(t.Context(), image.Image{ProductID: 7, ObjectKey: "products/7/a.png", Alt: "front"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	views, err := svc.URLsFor(t.Context(), 7)
	if err != nil {
		t.Fatalf("URLsFor: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("want 1 image, got %d", len(views))
	}
	if views[0].URL == "" {
		t.Fatal("want a non-empty presigned URL")
	}
	if strings.Contains(views[0].URL, "products/7/a.png") == false {
		t.Fatalf("URL should be derived from the object key, got %q", views[0].URL)
	}
}

// A stored URL outlives its TTL and starts failing for reasons nobody can
// see; presigning per call is what TestProductDetailCarriesPresignedURLs
// alone cannot prove.
func TestPresignedURLsAreGeneratedPerRequest(t *testing.T) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	svc := newTestService(store, repo, 1<<20)

	if _, err := repo.Create(t.Context(), image.Image{ProductID: 3, ObjectKey: "products/3/a.png"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, err := svc.URLsFor(t.Context(), 3)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := svc.URLsFor(t.Context(), 3)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if first[0].URL == second[0].URL {
		t.Fatal("want a fresh URL each call, got the same one twice")
	}
}
