package controllers

import (
	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/config"
)

// Package state until Tasks 6-7 convert these handlers to closures over
// *config.Config. Defaults are the safe values, not zero values.
var (
	hasher       = auth.NewHasher(auth.DefaultCost)
	cookieSecure = false
)

func Configure(cfg *config.Config) {
	hasher = cfg.Hasher()
	cookieSecure = cfg.IsProduction()
}
