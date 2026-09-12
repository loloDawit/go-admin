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

	name := path.Base(decodeBody(t, resp)["url"])
	if !storedNamePattern.MatchString(name) {
		t.Fatalf("stored name must be random hex plus extension, got %q", name)
	}
	if _, err := os.Stat(filepath.Join(testutil.TestConfig().UploadDir, name)); err != nil {
		t.Fatalf("file must exist under UploadDir as %q: %v", name, err)
	}
}

func decodeBody(t *testing.T, resp *http.Response) map[string]string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return body
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
	if code := decodeBody(t, resp)["code"]; code != "upload_unsupported_type" {
		t.Fatalf(`want code "upload_unsupported_type", got %q`, code)
	}
}

// Proves sniffing, not header-trust-plus-extension-check: both the filename
// (photo.png) and the Content-Type (image/png) claim a valid image, so only
// inspecting the actual bytes can catch the PHP payload underneath.
func TestUploadRejectsNonImageEvenWithLyingFilenameAndContentType(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "photo.png", "image/png", []byte("<?php system($_GET['c']); ?>"), cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a sniffed-non-image upload with a lying filename and Content-Type, got %d", resp.StatusCode)
	}
	if code := decodeBody(t, resp)["code"]; code != "upload_unsupported_type" {
		t.Fatalf(`want code "upload_unsupported_type", got %q`, code)
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
	// Pins the 413 to the handler's own MaxUploadBytes check: fasthttp's
	// BodyLimit also answers 413, and would do so silently if BodyLimit's
	// headroom over MaxUploadBytes ever collapsed to zero.
	if code := decodeBody(t, resp)["code"]; code != "upload_too_large" {
		t.Fatalf(`want code "upload_too_large", got %q`, code)
	}
}

func TestUploadGivesDistinctStoredNamesForSameClientFilename(t *testing.T) {
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
	url1 := decodeBody(t, resp1)["url"]

	req2 := uploadRequest(t, "dup.png", "image/png", secondBody, cookie)
	resp2, err := app.Test(req2, -1)
	if err != nil {
		t.Fatalf("request 2: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("want 200 on second upload, got %d", resp2.StatusCode)
	}
	url2 := decodeBody(t, resp2)["url"]

	if url1 == url2 {
		t.Fatalf("two uploads of the same client filename got the same stored name: %q", url1)
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

	url := decodeBody(t, resp)["url"]
	want := testutil.TestConfig().PublicBaseURL
	if !strings.HasPrefix(url, want) {
		t.Fatalf("url must start with the configured base %q, got %q", want, url)
	}
}

// Exercises all four allowlist entries, not just PNG: a regression dropping
// image/webp (say) would fail nothing otherwise.
func TestUploadAcceptsEachAllowedType(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	cases := []struct {
		name    string
		content []byte
		wantExt string
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0, 0, 0, 0, 0}, ".jpg"},
		{"png", pngBytes, ".png"},
		{"gif", []byte("GIF89a" + strings.Repeat("\x00", 6)), ".gif"},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), ".webp"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := uploadRequest(t, "upload"+c.wantExt, "application/octet-stream", c.content, cookie)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("want 200, got %d", resp.StatusCode)
			}
			url := decodeBody(t, resp)["url"]
			if !strings.HasSuffix(url, c.wantExt) {
				t.Fatalf("want stored name ending %q, got %q", c.wantExt, url)
			}
		})
	}
}

// A polyglot (valid GIF header, HTML/script tail) sniffs as an image and is
// stored — accepting it is not the defence; serving it with nosniff is,
// since fasthttp's static handler serves by extension regardless of content.
func TestUploadedFilesServeWithNosniffHeader(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	polyglot := append([]byte("GIF89a"), []byte("<script>alert(1)</script>")...)
	req := uploadRequest(t, "photo.gif", "image/gif", polyglot, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	url := decodeBody(t, resp)["url"]
	staticPath := strings.TrimPrefix(url, testutil.TestConfig().PublicBaseURL)

	getReq := httptest.NewRequest(http.MethodGet, staticPath, nil)
	getReq.AddCookie(cookie)
	getResp, err := app.Test(getReq, -1)
	if err != nil {
		t.Fatalf("static request: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 serving the stored file, got %d", getResp.StatusCode)
	}
	if got := getResp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf(`want X-Content-Type-Options: nosniff, got %q`, got)
	}
}

func TestMain(m *testing.M) {
	os.MkdirAll(testutil.TestConfig().UploadDir, 0o755)
	code := m.Run()
	os.RemoveAll("./testdata")
	os.Exit(code)
}
