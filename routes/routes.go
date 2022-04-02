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

	app.Post("/api/v1/logout", controllers.Logout)
	app.Get("/api/v1/user", controllers.User)

	app.Get("/api/v1/users", controllers.GetAllUsers)
	app.Post("/api/v1/users", controllers.CreateUser)
	app.Get("/api/v1/user/:id", controllers.GetUser)
	app.Put("/api/v1/user/:id", controllers.UpdateUser)
	app.Delete("/api/v1/user/:id", controllers.DeleteUser)
	// roles
	app.Get("/api/v1/roles", controllers.GetAllRoles)
	app.Post("/api/v1/roles", controllers.CreateRole)
	app.Get("/api/v1/role/:id", controllers.GetRole)
	app.Put("/api/v1/role/:id", controllers.UpdateRole)
	app.Delete("/api/v1/role/:id", controllers.DeleteRole)
}
