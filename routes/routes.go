package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/controllers"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/middlewares"
)

func SetupRoutes(app *fiber.App, cfg *config.Config) {
	controllers.Configure(cfg)

	api := app.Group("/api/v1")

	// No /register by design; see docs/decisions/0001-remove-public-registration.md.
	api.Post("/login", controllers.Login)

	authed := api.Group("", middlewares.IsAuthenticated)

	// Self-service: any signed-in user may read and edit their own account.
	// These need no resource permission — the handler scopes to the caller.
	authed.Get("/user", controllers.User)
	authed.Post("/logout", controllers.Logout)
	authed.Put("/user/info", controllers.UpdateUserInfo)
	authed.Put("/user/password", controllers.UpdatePassword)

	// Attached per route, not via a sibling Group(""): Fiber stacks a second
	// Group("", mw) onto every route sharing that empty prefix instead of
	// scoping it to its own routes.
	users := middlewares.RequirePermission("users")
	authed.Get("/users", users, controllers.GetAllUsers)
	authed.Post("/users", users, controllers.CreateUser)
	authed.Get("/user/:id", users, controllers.GetUser)
	authed.Put("/user/:id", users, controllers.UpdateUser)
	authed.Delete("/user/:id", users, controllers.DeleteUser)

	products := middlewares.RequirePermission("products")
	authed.Get("/products", products, controllers.GetAllProducts)
	authed.Post("/products", products, controllers.CreateProduct)
	authed.Get("/product/:id", products, controllers.GetProduct)
	authed.Put("/product/:id", products, controllers.UpdateProduct)
	authed.Delete("/product/:id", products, controllers.DeleteProduct)
	authed.Post("/upload", products, controllers.Upload(cfg))

	orders := middlewares.RequirePermission("orders")
	authed.Get("/orders", orders, controllers.GetAllOrders)
	authed.Post("/orders", orders, controllers.CreateOrder)
	authed.Get("/order/:id", orders, controllers.GetOrder)
	authed.Put("/order/:id", orders, controllers.UpdateOrder)
	authed.Delete("/order/:id", orders, controllers.DeleteOrder)
	authed.Get("/export", orders, controllers.Export)
	authed.Get("/chart", orders, controllers.Chart)

	roles := middlewares.RequirePermission("roles")
	authed.Get("/roles", roles, controllers.GetAllRoles)
	authed.Post("/roles", roles, controllers.CreateRole)
	authed.Get("/role/:id", roles, controllers.GetRole)
	authed.Put("/role/:id", roles, controllers.UpdateRole)
	authed.Delete("/role/:id", roles, controllers.DeleteRole)
	authed.Get("/permissions", roles, controllers.GetAllPermissions)
	authed.Post("/permissions", roles, controllers.CreatePermission)

	// Uploaded files.
	app.Static("/api/v1/uploads", cfg.UploadDir)
}
