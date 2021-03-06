package routes

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/controllers"
	"gitlab.nordstrom.com/go-admin/middlewares"
)

func SetupRoutes(app *fiber.App) {
	app.Post("/api/v1/register", controllers.Register)
	app.Post("/api/v1/login", controllers.Login)

	// private routes
	app.Use(middlewares.IsAuthenticated)
	// Auth routes
	app.Put("/api/v1/user/info", controllers.UpdateUserInfo)
	app.Put("/api/v1/user/password", controllers.UpdatePassword)
	app.Post("/api/v1/logout", controllers.Logout)
	app.Get("/api/v1/user", controllers.User)

	// users routes
	app.Get("/api/v1/users", controllers.GetAllUsers)
	app.Post("/api/v1/users", controllers.CreateUser)
	app.Get("/api/v1/user/:id", controllers.GetUser)
	app.Put("/api/v1/user/:id", controllers.UpdateUser)
	app.Delete("/api/v1/user/:id", controllers.DeleteUser)

	// products routes
	app.Get("/api/v1/products", controllers.GetAllProducts)
	app.Post("/api/v1/products", controllers.CreateProduct)
	app.Get("/api/v1/product/:id", controllers.GetProduct)
	app.Put("/api/v1/product/:id", controllers.UpdateProduct)
	app.Delete("/api/v1/product/:id", controllers.DeleteProduct)

	// order routes
	app.Get("/api/v1/orders", controllers.GetAllOrders)
	app.Post("/api/v1/orders", controllers.CreateOrder)
	app.Get("/api/v1/order/:id", controllers.GetOrder)
	app.Put("/api/v1/order/:id", controllers.UpdateOrder)
	app.Delete("/api/v1/order/:id", controllers.DeleteOrder)
	app.Post("/api/v1/export", controllers.Export)
	app.Get("/api/v1/chart", controllers.Chart)

	// roles routes
	app.Get("/api/v1/roles", controllers.GetAllRoles)
	app.Post("/api/v1/roles", controllers.CreateRole)
	app.Get("/api/v1/role/:id", controllers.GetRole)
	app.Put("/api/v1/role/:id", controllers.UpdateRole)
	app.Delete("/api/v1/role/:id", controllers.DeleteRole)

	// permission routes
	app.Get("/api/v1/permissions", controllers.GetAllPermissions)
	app.Post("/api/v1/permissions", controllers.CreatPermissons)

	app.Post("/api/v1/upload", controllers.Upload)

	//static

	app.Static("/api/v1/uploads", "./uploads")

}
