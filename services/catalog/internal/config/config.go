package config

import (
	"errors"
	"os"
)

const serviceName = "catalog"

// DefaultPort is copied by every service derived from this template.
// serviceName above and DefaultPort are the two values in this file a
// service-specific rename touches.
const DefaultPort = "8082"

type Config struct {
	ServiceName string
	Port        string
	DatabaseURL string
}

// Any error here is fatal: a misconfigured service must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: serviceName,
		Port:        withDefault("PORT", DefaultPort),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
