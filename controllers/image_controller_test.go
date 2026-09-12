package controllers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
)

func uploadRequest(t *testing.T, filename, contentType string, content []byte, cookie *http.Cookie) *http.Request {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="image"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}

	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("multipart: %v", err)
	}
	part.Write(content)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)
	return req
}

// A PNG header, so the MIME sniffer sees a real image.
var pngBytes = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}

var storedNamePattern = regexp.MustCompile(`^[0-9a-f]{32}\.png$`)

func TestUploadRejectsPathTraversal(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "../../../../tmp/pwned.png", "image/png", pngBytes, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 for a valid image despite the traversal filename, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)

	name := path.Base(body["url"])
	if !storedNamePattern.MatchString(name) {
		t.Fatalf("stored name must be random hex plus extension, got %q", name)
	}
	if _, err := os.Stat(filepath.Join(testutil.TestConfig().UploadDir, name)); err != nil {
		t.Fatalf("file must exist under UploadDir as %q: %v", name, err)
	}
}

func TestUploadRejectsNonImage(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "shell.php", "application/x-php", []byte("<?php system($_GET['c']); ?>"), cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a non-image upload, got %d", resp.StatusCode)
	}
}

// Proves sniffing rather than trusting the header: a lying Content-Type of
// image/png on a PHP payload must still be rejected.
func TestUploadRejectsNonImageEvenWithLyingContentType(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "shell.php", "image/png", []byte("<?php system($_GET['c']); ?>"), cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a sniffed-non-image upload with a lying Content-Type, got %d", resp.StatusCode)
	}
}

func TestUploadRejectsOversized(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	limit := testutil.TestConfig().MaxUploadBytes
	oversized := make([]byte, limit+1)
	copy(oversized, pngBytes)

	req := uploadRequest(t, "big.png", "image/png", oversized, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413 for an oversized upload, got %d", resp.StatusCode)
	}
}

func TestUploadDoesNotOverwriteOnCollision(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	firstBody := append([]byte{}, pngBytes...)
	secondBody := append(append([]byte{}, pngBytes...), 0xAA, 0xBB)

	req1 := uploadRequest(t, "dup.png", "image/png", firstBody, cookie)
	resp1, err := app.Test(req1, -1)
	if err != nil {
		t.Fatalf("request 1: %v", err)
	}
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("want 200 on first upload, got %d", resp1.StatusCode)
	}
	var body1 map[string]string
	json.NewDecoder(resp1.Body).Decode(&body1)

	req2 := uploadRequest(t, "dup.png", "image/png", secondBody, cookie)
	resp2, err := app.Test(req2, -1)
	if err != nil {
		t.Fatalf("request 2: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("want 200 on second upload, got %d", resp2.StatusCode)
	}
	var body2 map[string]string
	json.NewDecoder(resp2.Body).Decode(&body2)

	if body1["url"] == body2["url"] {
		t.Fatalf("two uploads of the same client filename collided on the same stored name: %q", body1["url"])
	}
}

func TestUploadReturnsConfiguredBaseURL(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "photo.png", "image/png", pngBytes, cookie)
	resp, _ := app.Test(req, -1)

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)

	want := testutil.TestConfig().PublicBaseURL
	if !strings.HasPrefix(body["url"], want) {
		t.Fatalf("url must start with the configured base %q, got %q", want, body["url"])
	}
}

func TestMain(m *testing.M) {
	os.MkdirAll(testutil.TestConfig().UploadDir, 0o755)
	code := m.Run()
	os.RemoveAll("./testdata")
	os.Exit(code)
}
