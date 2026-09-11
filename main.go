package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/routes"
	"github.com/loloDawit/go-admin/utils"
)

func main() {
	// In production the platform supplies the environment, so a missing .env
	// is not an error.
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Printf("config: .env not loaded (%v); reading environment directly", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	utils.SecretKey = cfg.SessionSecret

	if _, err := database.Connect(cfg.DBDSN); err != nil {
		log.Fatalf("database: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: int(cfg.MaxUploadBytes) + (1 << 20), // upload cap plus headroom
	})

	// Must be an explicit origin: wildcard-with-credentials is a CSRF hole.
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigin,
		AllowCredentials: true,
		AllowHeaders:     "Content-Type",
	}))

	routes.SetupRoutes(app, cfg)

	log.Printf("listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
