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
	
	app.Get("/api/v1/logout", controllers.Logout)
	app.Get("/api/v1/user", controllers.User)
}
