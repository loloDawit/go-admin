// Package arch_test: not redundant with the compiler's internal/ rule, which only protects the current file layout.
package arch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/loloDawit/go-admin"

// httpxImportPath: spec §9 requires status codes and client messages to
// originate in exactly one place per service, the service's own internal/httperr.
const httpxImportPath = modulePath + "/platform/httpx"

var services = []string{"gateway", "identity", "catalog", "orders"}

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

// This scans file names, not identifiers or content: platform/observability's
// statusRecorder identifier contains "order", which a content scan would flag.
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

	checkPlatformHoldsNoPermissionVocabularyLiterals(t, root, platformDir)
}

// permissionVocabularyPattern matches Identity's permission-string shape (e.g. "view_staff"), catching a leak even when the file name gives no hint (spec §5).
var permissionVocabularyPattern = regexp.MustCompile(`^(view|edit)_[a-z_]+$`)

// Test files are scanned too: a fixture value is as much a leak as production code.
func checkPlatformHoldsNoPermissionVocabularyLiterals(t *testing.T, root, platformDir string) {
	t.Helper()

	scanned := 0
	err := filepath.Walk(platformDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		scanned++

		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", path, perr)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}
			if permissionVocabularyPattern.MatchString(value) {
				pos := fset.Position(lit.Pos())
				t.Errorf("%s:%d contains permission-vocabulary string literal %q.\n\tplatform/ is technical infrastructure only; this belongs under services/identity/internal instead.", rel(root, path), pos.Line, value)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk platform for permission-vocabulary literals: %v", err)
	}
	if scanned == 0 {
		t.Fatalf("no .go files found under %s; this guard would otherwise pass vacuously", rel(root, platformDir))
	}
}

// Resolves each file's local import alias rather than string-matching "httpx.WriteError", so an aliased import cannot dodge the check.
func TestClientFacingErrorsOnlyConstructedInHTTPErr(t *testing.T) {
	root := repoRoot(t)

	for _, svc := range services {
		svc := svc
		t.Run(svc, func(t *testing.T) {
			svcDir := filepath.Join(root, "services", svc)
			httperrDir := filepath.Join(svcDir, "internal", "httperr") + string(filepath.Separator)

			callsInHTTPErr := 0

			err := filepath.Walk(svcDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return err
				}

				fset := token.NewFileSet()
				file, perr := parser.ParseFile(fset, path, nil, 0)
				if perr != nil {
					t.Fatalf("parse %s: %v", path, perr)
				}

				alias := httpxLocalAlias(file)
				if alias == "" {
					return nil // this file does not import platform/httpx at all
				}

				inHTTPErr := strings.HasPrefix(path, httperrDir)

				ast.Inspect(file, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "WriteError" {
						return true
					}
					ident, ok := sel.X.(*ast.Ident)
					if !ok || ident.Name != alias {
						return true
					}

					if inHTTPErr {
						callsInHTTPErr++
						return true
					}
					pos := fset.Position(call.Pos())
					t.Errorf("%s:%d constructs a client-facing error via httpx.WriteError outside %sinternal/httperr.\n\tStatus codes and client messages must originate from exactly one place per service; call into internal/httperr instead.",
						rel(root, path), pos.Line, svc+"/")
					return true
				})
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", svcDir, err)
			}
			if callsInHTTPErr == 0 {
				t.Fatalf("found zero httpx.WriteError calls inside %s; an AST-shape mismatch would otherwise make this guard pass vacuously", rel(root, strings.TrimSuffix(httperrDir, string(filepath.Separator))))
			}
		})
	}
}

// httpxLocalAlias returns "" if file does not import platform/httpx at all.
func httpxLocalAlias(file *ast.File) string {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path != httpxImportPath {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		parts := strings.Split(path, "/")
		return parts[len(parts)-1]
	}
	return ""
}

// forEachImport parses only the import block, so it works without a database or Docker.
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

// hasPathPrefix matches on full path segments: a bare strings.HasPrefix would
// let prefix ".../internal" match ".../internalfoo".
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
