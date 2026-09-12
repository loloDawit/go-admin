// Package arch_test enforces import boundaries that Go's own internal/ rule
// already enforces today, at compile time. That mechanism only protects the
// current file layout: someone can resolve a future compile error by moving a
// package out of internal/ (or copying code across a boundary) instead of by
// respecting the boundary, and the compiler will not object. These tests exist
// to catch that regression, so do not delete them as "redundant with the
// compiler" — the compiler check they duplicate is not the check they perform.
package arch_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/loloDawit/go-admin"

var services = []string{"gateway", "identity", "catalog", "orders"}

// Legacy packages live at the repo root and are not part of the new platform
// layout. Nothing under services/ or platform/ may depend on them: doing so
// would tie the new code's compilation to code that lives outside this
// boundary. This list must stay a superset of every top-level legacy
// directory, including cmd/seed (a subdirectory of the new cmd/ tree, not a
// service) and clients/.
var legacyRoots = []string{
	modulePath + "/controllers",
	modulePath + "/models",
	modulePath + "/routes",
	modulePath + "/middlewares",
	modulePath + "/database",
	modulePath + "/utils",
	modulePath + "/clients",
	modulePath + "/internal", // the root's internal/, not platform/'s or a service's
	modulePath + "/cmd/seed",
}

func TestNoServiceImportsAnotherService(t *testing.T) {
	root := repoRoot(t)

	for _, svc := range services {
		svc := svc
		t.Run(svc, func(t *testing.T) {
			forEachImport(t, filepath.Join(root, "services", svc), func(file, imported string) {
				for _, other := range services {
					if other == svc {
						continue
					}
					if hasPathPrefix(imported, modulePath+"/services/"+other) {
						t.Errorf("%s imports %s.\n\tServices communicate over HTTP through published contracts, never by importing each other. Remove this import.", rel(root, file), imported)
					}
				}
			})
		})
	}
}

func TestNoNewCodeImportsLegacyPackages(t *testing.T) {
	root := repoRoot(t)

	for _, dir := range []string{"services", "platform"} {
		forEachImport(t, filepath.Join(root, dir), func(file, imported string) {
			for _, legacy := range legacyRoots {
				if hasPathPrefix(imported, legacy) {
					t.Errorf("%s imports legacy package %s.\n\tNew code must not depend on the legacy application at the repo root; use the corresponding platform/ or services/*/internal package instead.", rel(root, file), imported)
				}
			}
		})
	}
}

// The check below scans file names, not identifiers or file content. That is
// deliberate: platform/observability legitimately defines identifiers like
// statusRecorder and a records field, both of which contain the substring
// "order" — a content or identifier scan would fail on correct code. A file
// name containing a domain word is a much stronger signal that domain logic
// leaked into platform/, so this guard trades recall for zero false
// positives on the technical vocabulary platform/ actually needs.
func TestPlatformHoldsNoDomainConcepts(t *testing.T) {
	root := repoRoot(t)
	platformDir := filepath.Join(root, "platform")

	if _, err := os.Stat(platformDir); os.IsNotExist(err) {
		t.Fatalf("platform/ does not exist at %s; this guard cannot run", platformDir)
	}

	banned := []string{"product", "order", "customer", "staff", "permission", "role", "session"}

	err := filepath.Walk(platformDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		name := strings.ToLower(filepath.Base(path))
		for _, word := range banned {
			if strings.Contains(name, word) {
				t.Errorf("%s looks like a domain concept.\n\tplatform/ is technical infrastructure only; move this file under services/*/internal instead.", rel(root, path))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk platform: %v", err)
	}
}

// forEachImport parses only the import block of every .go file under dir (no
// type-checking, so it works without a database or Docker) and calls check
// once per import. It fails the test outright if dir is missing or contains
// no .go files: a silent no-op here would let every guard above pass
// vacuously if services/ or platform/ were ever renamed.
func forEachImport(t *testing.T, dir string, check func(file, imported string)) {
	t.Helper()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("%s does not exist; this guard has nothing to check", dir)
	}

	scanned := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		scanned++
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			check(path, strings.Trim(imp.Path.Value, `"`))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	if scanned == 0 {
		t.Fatalf("no .go files found under %s; this guard would otherwise pass vacuously", dir)
	}
}

// hasPathPrefix reports whether imported is prefix or a subpackage of it,
// matching on full path segments so that, e.g., the legacy root's
// .../internal/httpx does not also match .../platform/httpx or
// .../services/gateway/internal/httpx: both share the "internal" trailing
// segment as a substring but neither is the root package legacyRoots names.
func hasPathPrefix(imported, prefix string) bool {
	return imported == prefix || strings.HasPrefix(imported, prefix+"/")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate the repository root")
	return ""
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return r
}
