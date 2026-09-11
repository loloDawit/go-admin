package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/models"
	"github.com/loloDawit/go-admin/routes"
	"github.com/loloDawit/go-admin/utils"
	"gorm.io/gorm"
)

func TestConfig() *config.Config {
	return &config.Config{
		DBDSN:          "unused-in-tests",
		SessionSecret:  "0123456789abcdef0123456789abcdef",
		Port:           "0",
		AllowedOrigin:  "http://localhost:3000",
		PublicBaseURL:  "http://localhost:8080",
		UploadDir:      "./testdata/uploads",
		MaxUploadBytes: 5 << 20,
		AppEnv:         "test",
		BcryptCost:     auth.NewTestHasher().Cost(),
	}
}

// NewApp builds the real routed application: same routes, same middleware
// chain, no test-only wiring.
func NewApp(t *testing.T) *fiber.App {
	t.Helper()

	cfg := TestConfig()
	utils.SecretKey = cfg.SessionSecret

	app := fiber.New()
	routes.SetupRoutes(app, cfg)
	return app
}

func NewRequest(method, target string, body io.Reader, cookie *http.Cookie) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

func JSON(s string) io.Reader { return strings.NewReader(s) }

// The password goes through the real hasher so login tests are honest.
func SeedUser(t *testing.T, db *gorm.DB, email, password, roleName string) *models.User {
	t.Helper()

	role := SeedRole(t, db, roleName)

	hash, err := auth.NewTestHasher().Hash(password)
	if err != nil {
		t.Fatalf("testutil: hash password: %v", err)
	}

	user := &models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     email,
		Password:  hash,
		RoleId:    role.Id,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("testutil: seed user: %v", err)
	}
	return user
}

func SeedRole(t *testing.T, db *gorm.DB, name string) *models.Role {
	t.Helper()

	var role models.Role
	if err := db.Where(models.Role{Name: name}).
		FirstOrCreate(&role, models.Role{Name: name}).Error; err != nil {
		t.Fatalf("testutil: seed role %q: %v", name, err)
	}
	return &role
}

func GrantPermission(t *testing.T, db *gorm.DB, roleName, permission string) {
	t.Helper()

	role := SeedRole(t, db, roleName)

	var perm models.Permission
	if err := db.Where(models.Permission{Name: permission}).
		FirstOrCreate(&perm, models.Permission{Name: permission}).Error; err != nil {
		t.Fatalf("testutil: seed permission %q: %v", permission, err)
	}
	if err := db.Model(role).Association("Permissions").Append(&perm); err != nil {
		t.Fatalf("testutil: grant %q to %q: %v", permission, roleName, err)
	}
}

func Login(t *testing.T, app *fiber.App, email, password string) *http.Cookie {
	t.Helper()

	req := NewRequest(http.MethodPost, "/api/v1/login",
		JSON(`{"email":"`+email+`","password":"`+password+`"}`), nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("testutil: login request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("testutil: login failed with status %d", resp.StatusCode)
	}

	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			return c
		}
	}
	t.Fatal("testutil: login response carried no jwt cookie")
	return nil
}
