package controllers

import (
	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/config"
)

// hasher is the password hasher these handlers use.
//
// It is package state, which is a wart — but a deliberate, bounded one: the
// handlers in this package are still plain functions inherited from the
// original code, so there is nowhere to inject a dependency yet. Tasks 4, 6
// and 7 convert the handlers that need configuration into closures over
// *config.Config, at which point this variable and Configure go away.
//
// It is initialised to the safe default rather than left as a zero value, so
// a handler reached before Configure still hashes at DefaultCost instead of
// silently using bcrypt's minimum.
var hasher = auth.NewHasher(auth.DefaultCost)

// cookieSecure marks the session cookie HTTPS-only. It is false by default so
// that local development over plain HTTP still works; Configure turns it on
// from APP_ENV.
var cookieSecure = false

// Configure wires package-level dependencies from configuration. SetupRoutes
// calls it, so any process that serves routes is configured by construction.
func Configure(cfg *config.Config) {
	hasher = cfg.Hasher()
	cookieSecure = cfg.IsProduction()
}
