package permission_test

import (
	"os"
	"slices"
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"gopkg.in/yaml.v3"
)

// openAPIDoc reaches components.schemas.Permission.enum; yaml.v3 ignores any
// key with no matching struct field, so the rest of identity.yaml is not
// modeled here.
type openAPIDoc struct {
	Components struct {
		Schemas struct {
			Permission struct {
				Enum []string `yaml:"enum"`
			} `yaml:"Permission"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

// TestAllMatchesPublishedOpenAPIEnum guards against a permission added to Go
// but never published: without this, that gap would only surface in M3, when
// Catalog or Orders generates constants from an OpenAPI spec missing one.
func TestAllMatchesPublishedOpenAPIEnum(t *testing.T) {
	data, err := os.ReadFile("../../openapi/identity.yaml")
	if err != nil {
		t.Fatalf("read identity.yaml: %v", err)
	}

	var doc openAPIDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse identity.yaml: %v", err)
	}

	got := slices.Clone(doc.Components.Schemas.Permission.Enum)
	want := permission.All()

	slices.Sort(got)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("identity.yaml Permission enum differs from permission.All():\n yaml: %v\n go:   %v", got, want)
	}
}
