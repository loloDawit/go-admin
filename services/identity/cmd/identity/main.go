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
	pgxplatform "github.com/loloDawit/go-admin/platform/pgx"
	"github.com/loloDawit/go-admin/platform/readiness"
	"github.com/loloDawit/go-admin/services/identity/internal/config"
	"github.com/loloDawit/go-admin/services/identity/internal/httperr"
	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"github.com/loloDawit/go-admin/services/identity/internal/role"
	"github.com/loloDawit/go-admin/services/identity/internal/schemacheck"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
	"github.com/loloDawit/go-admin/services/identity/internal/staff"
)

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

	logger := observability.NewLogger(cfg.ServiceName, os.Stdout)
	slog.SetDefault(logger)

	ctx := context.Background()

	shutdownTracing, err := observability.NewTracerProvider(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("tracing", slog.String("error", err.Error()))
		os.Exit(1)
	}

	shutdownMetrics, err := observability.NewMeterProvider(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// A database that is down must not stop the process from starting; readiness
	// reports it and the service recovers when Postgres returns.
	pool, err := pgxplatform.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}

	errWriter := httperr.New(logger)
	svc := schemacheck.NewService(schemacheck.NewPostgresRepository(pool))

	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	sessionSvc, err := session.NewService(session.NewPostgresRepository(pool), session.NewHasher(cfg.BcryptCost), cfg.SessionTTL)
	if err != nil {
		logger.Error("session service", slog.String("error", err.Error()))
		os.Exit(1)
	}
	sessionHandler := session.NewHandler(sessionSvc, cfg.CookieSecure, cfg.SessionTTL, cfg.MaxRequestBodyBytes, errWriter.Write)

	staffSvc := staff.NewService(staff.NewPostgresRepository(pool), session.NewHasher(cfg.BcryptCost), cfg.PageSizeMax)
	staffHandler := staff.NewHandler(staffSvc, cfg.MaxRequestBodyBytes, errWriter.Write)

	roleSvc := role.NewService(role.NewPostgresRepository(pool), cfg.PageSizeMax)
	roleHandler := role.NewHandler(roleSvc, cfg.MaxRequestBodyBytes, errWriter.Write)

	permissionHandler := permission.NewHandler()

	r := newRouter(logger, ready, sessionHandler, staffHandler, roleHandler, permissionHandler, cfg.PrincipalKey, errWriter.Write)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// A listener failure reports through serverErr rather than os.Exit in the
	// goroutine, so the shutdown path below (and pool.Close) still runs.
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
	pool.Close()

	os.Exit(exitCode)
}
