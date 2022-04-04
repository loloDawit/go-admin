package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"gitlab.nordstrom.com/go-admin/database"
	"gitlab.nordstrom.com/go-admin/routes"
	"gitlab.nordstrom.com/go-admin/utils"
)

func main() {
	// Load env files

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	utils.SecretKey = os.Getenv("SECRET")

	// Create db connection
	database.Connect()

	// setup routes
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	routes.SetupRoutes(app)

	app.Listen(":8080")
}
