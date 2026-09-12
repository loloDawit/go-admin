package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/services/gateway/internal/config"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
)

const serviceName = "gateway"

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := observability.NewLogger(serviceName, os.Stdout)
	slog.SetDefault(logger)

	upstreams, err := routing.New(cfg.Upstreams, cfg.UpstreamTimeout)
	if err != nil {
		logger.Error("routing", slog.String("error", err.Error()))
		os.Exit(1)
	}

	r := newRouter(logger, upstreams)

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
	cancel()

	os.Exit(exitCode)
}
