package main

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/routes"
)

func main() {
	// Create db connection
	database.Connect()

	// setup routes
	app := fiber.New()

	routes.SetupRoutes(app)

	app.Listen(":3000")
}
