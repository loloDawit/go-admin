// Command seed bootstraps a database with the permission vocabulary, the
// built-in roles, and the owner account. It is idempotent — safe to run on
// every deploy.
//
//	OWNER_EMAIL=you@example.com OWNER_PASSWORD=... go run ./cmd/seed
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/internal/seed"
)

func main() {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Printf("config: .env not loaded (%v); reading environment directly", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DBDSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	ownerEmail := os.Getenv("OWNER_EMAIL")
	ownerPassword := os.Getenv("OWNER_PASSWORD")

	if err := seed.Run(db, cfg.Hasher(), ownerEmail, ownerPassword); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("seeded: permissions, roles, and owner %s", ownerEmail)
	log.Print("change the owner password after first sign-in, then remove OWNER_PASSWORD from .env")
}
