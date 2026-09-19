package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/loloDawit/go-admin/platform/healthcheck"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
	"github.com/loloDawit/go-admin/services/gateway/internal/config"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
	"github.com/loloDawit/go-admin/services/gateway/internal/web"
)

const serviceName = "gateway"

func main() {
	// Must exit before any normal startup below runs a second copy of the process.
	healthCheck := flag.Bool("health-check", false, "check this process's own /healthz and exit 0 or 1")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if *healthCheck {
		os.Exit(healthcheck.Run(cfg.Port))
	}

	logger := observability.NewLogger(serviceName, os.Stdout)
	slog.SetDefault(logger)

	ctx := context.Background()
	shutdownTracing, err := observability.NewTracerProvider(ctx, serviceName, cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("tracing", slog.String("error", err.Error()))
		os.Exit(1)
	}

	shutdownMetrics, err := observability.NewMeterProvider(ctx, serviceName, cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var spa http.Handler
	if cfg.WebRoot != "" {
		spa, err = web.Handler(cfg.WebRoot)
		if err != nil {
			logger.Error("web root", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("serving the frontend", slog.String("web_root", cfg.WebRoot))
	}

	upstreams, err := routing.New(logger, cfg.Upstreams, cfg.UpstreamTimeout, spa)
	if err != nil {
		logger.Error("routing", slog.String("error", err.Error()))
		os.Exit(1)
	}

	validator := auth.NewValidator(
		&http.Client{Timeout: cfg.UpstreamTimeout},
		cfg.Upstreams["identity"]+"/internal/sessions/validate",
		cfg.PrincipalKey,
		cfg.PrincipalTTL,
		auth.NewCache(cfg.SessionCacheTTL),
		httperr.New(logger),
		logger,
	)
	logger.Info("session validation configured",
		slog.Duration("principal_ttl", cfg.PrincipalTTL),
		slog.Duration("session_cache_ttl", cfg.SessionCacheTTL),
	)

	r := newRouter(logger, upstreams, validator)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// A listener failure reports through serverErr rather than os.Exit in the
	// goroutine, so the shutdown path below still runs.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	select {
	case <-stop:
	case err := <-serverErr:
		if err != nil {
			logger.Error("server", slog.String("error", err.Error()))
			exitCode = 1
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = srv.Shutdown(shutdownCtx)
	_ = shutdownTracing(shutdownCtx)
	_ = shutdownMetrics(shutdownCtx)
	cancel()

	os.Exit(exitCode)
}
