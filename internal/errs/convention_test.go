package errs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// apiPackages are the layers whose errors can reach a client. Every failure
// they produce must come from the errs registry, so that its code, message,
// and status are declared in exactly one place and no handler can invent a
// message the frontend has never seen.
var apiPackages = []string{"controllers", "models", "middlewares", "routes"}

// infrastructurePackages legitimately use fmt.Errorf. Their errors are
// operator-facing startup diagnostics read once at boot (config validation,
// database connection) — they never reach an HTTP response, so a stable
// machine-readable code would be ceremony with no consumer.
//
// This test exists to keep that boundary honest: if one of these ever starts
// serving an error to a client, it belongs in the registry instead.
var infrastructurePackages = []string{"internal/config", "database", "internal/seed", "cmd"}

// TestNoInlineErrorsInAPILayers fails if an API-layer file constructs an error
// with errors.New or fmt.Errorf instead of returning a registry value.
func TestNoInlineErrorsInAPILayers(t *testing.T) {
	root := repoRoot(t)

	for _, pkg := range apiPackages {
		dir := filepath.Join(root, pkg)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			checkFile(t, root, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", pkg, err)
		}
	}
}

func checkFile(t *testing.T, root, path string) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	rel, _ := filepath.Rel(root, path)

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		banned := (ident.Name == "errors" && sel.Sel.Name == "New") ||
			(ident.Name == "fmt" && sel.Sel.Name == "Errorf")
		if !banned {
			return true
		}

		t.Errorf("%s:%d: %s.%s is not allowed in an API layer.\n"+
			"\tDeclare the failure in internal/errs/registry.go and return that value,\n"+
			"\tso its code, message, and HTTP status live in one place.\n"+
			"\tTo attach a cause: errs.Something.Wrap(err)",
			rel, fset.Position(call.Pos()).Line, ident.Name, sel.Sel.Name)
		return true
	})
}

// Documents the boundary: infrastructure packages are deliberately exempt.
func TestInfrastructurePackagesAreDocumentedExemptions(t *testing.T) {
	root := repoRoot(t)

	for _, pkg := range infrastructurePackages {
		if _, err := os.Stat(filepath.Join(root, pkg)); os.IsNotExist(err) {
			continue // not built yet
		}
		for _, api := range apiPackages {
			if pkg == api {
				t.Errorf("%s cannot be both an API layer and an infrastructure exemption", pkg)
			}
		}
	}
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
