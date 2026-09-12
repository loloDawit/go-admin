package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	Port            string
	Upstreams       map[string]string
	UpstreamTimeout time.Duration
}

// Any error here is fatal: a misconfigured gateway must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            withDefault("PORT", "8080"),
		UpstreamTimeout: 5 * time.Second,
		Upstreams: map[string]string{
			"identity": os.Getenv("IDENTITY_URL"),
			"catalog":  os.Getenv("CATALOG_URL"),
			"orders":   os.Getenv("ORDERS_URL"),
		},
	}
	for name, url := range cfg.Upstreams {
		if url == "" {
			return nil, errors.New("upstream URL is required for " + name)
		}
	}
	return cfg, nil
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
