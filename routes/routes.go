package routes

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/controllers"
)

func SetupRoutes(app *fiber.App) {

	app.Get("/", controllers.Hello)
}
