package image_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/services/catalog/internal/image"
)

func withChiURLParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func newTestHandler(maxBytes int64) (*image.Handler, *fakeStore, *fakeRepository, *error) {
	store := newFakeStore()
	repo := newFakeRepository(nil)
	svc := image.NewService(repo, store, maxBytes)
	var captured error
	h := image.NewHandler(svc, maxBytes, func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusInternalServerError)
	})
	return h, store, repo, &captured
}

// multipartUploadRequest builds a real HTTP request, filename and all: the
// point is proving the handler never lets that filename reach a path.
func multipartUploadRequest(t *testing.T, filename string, content []byte, declaredType string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="` + filename + `"`},
		"Content-Type":        {declaredType},
	})
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/42/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = withChiURLParams(req, map[string]string{"id": "42"})
	return req
}

// A multipart filename of "../../etc/passwd" must be impossible to exploit
// because the object key is never built from it, not because it was
// sanitised: the resulting key must still be products/{id}/{uuid}.{ext}.
func TestUploadHandlerObjectKeyIgnoresATraversalFilename(t *testing.T) {
	h, _, repo, captured := newTestHandler(1 << 20)
	req := multipartUploadRequest(t, "../../etc/passwd", validPNGBytes(64), "image/png")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if *captured != nil {
		t.Fatalf("want success, got %v", *captured)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.byID) != 1 {
		t.Fatalf("want exactly one image row, got %d", len(repo.byID))
	}
	var stored image.Image
	for _, v := range repo.byID {
		stored = v
	}
	if strings.Contains(stored.ObjectKey, "..") || strings.Contains(stored.ObjectKey, "etc") || strings.Contains(stored.ObjectKey, "passwd") {
		t.Fatalf("object key leaked the client filename: %q", stored.ObjectKey)
	}

	var resp image.ImageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.Contains(resp.URL, "passwd") {
		t.Fatalf("response leaked the client filename via the URL: %q", resp.URL)
	}
}

// A declared Content-Type of image/png over bytes that are not actually a
// PNG must be refused, surfacing ErrUnsupportedImageType through the
// injected writeErr rather than a generic 500.
func TestUploadHandlerRejectsADeclaredTypeThatLies(t *testing.T) {
	h, store, _, captured := newTestHandler(1 << 20)
	req := multipartUploadRequest(t, "photo.png", []byte("#!/bin/sh\necho pwned\n"), "image/png")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a declared type that does not match the sniffed bytes")
	}
	if store.putCalled {
		t.Fatal("Put must not be called once the sniff rejects the upload")
	}
}

// Content-Length over the cap must be refused before the handler reads
// anything from the body.
func TestUploadHandlerRefusesABodyOverTheCap(t *testing.T) {
	h, store, _, captured := newTestHandler(10)
	req := multipartUploadRequest(t, "photo.png", validPNGBytes(1000), "image/png")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a body over the configured cap")
	}
	if store.putCalled {
		t.Fatal("Put must not be called once the cap rejects the upload")
	}
}

func TestDeleteHandlerRemovesTheImage(t *testing.T) {
	h, store, repo, captured := newTestHandler(1 << 20)
	created, err := repo.Create(t.Context(), image.Image{ProductID: 42, ObjectKey: "products/42/a.png"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/42/images/"+strconv.FormatInt(created.ID, 10), nil)
	req = withChiURLParams(req, map[string]string{"id": "42", "imageID": strconv.FormatInt(created.ID, 10)})
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if *captured != nil {
		t.Fatalf("want success, got %v", *captured)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: want 204, got %d", rec.Code)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "products/42/a.png" {
		t.Fatalf("want the object deleted, got %v", store.deleted)
	}
	if _, ok := repo.byID[created.ID]; ok {
		t.Fatal("want the row deleted")
	}
}

func TestDeleteHandlerRefusesANonNumericImageID(t *testing.T) {
	h, _, _, captured := newTestHandler(1 << 20)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/42/images/not-a-number", nil)
	req = withChiURLParams(req, map[string]string{"id": "42", "imageID": "not-a-number"})
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric image id")
	}
}
