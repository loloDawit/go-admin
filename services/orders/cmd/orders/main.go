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
	"github.com/loloDawit/go-admin/services/orders/internal/catalog"
	"github.com/loloDawit/go-admin/services/orders/internal/config"
	"github.com/loloDawit/go-admin/services/orders/internal/customer"
	"github.com/loloDawit/go-admin/services/orders/internal/httperr"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
	"github.com/loloDawit/go-admin/services/orders/internal/reporting"
	"github.com/loloDawit/go-admin/services/orders/internal/schemacheck"
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

	catalogClient := catalog.NewClient(cfg.CatalogURL, cfg.CatalogTimeout, cfg.CatalogMaxIdleConns)

	customerSvc := customer.NewService(customer.NewPostgresRepository(pool), cfg.OrderPageSizeMax)
	customerHandler := customer.NewHandler(customerSvc, cfg.MaxRequestBodyBytes, errWriter.Write)

	orderSvc := order.NewService(order.NewPostgresRepository(pool), catalogClient, cfg.OrderPageSizeMax)
	orderHandler := order.NewHandler(orderSvc, cfg.MaxRequestBodyBytes, errWriter.Write)

	reportingSvc := reporting.NewService(reporting.NewPostgresRepository(pool), cfg.ReportWindowDays)
	reportingHandler := reporting.NewHandler(reportingSvc, errWriter.Write)

	r := newRouter(logger, ready, customerHandler, orderHandler, reportingHandler, cfg.PrincipalKey, errWriter.Write)

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
	cancel()
	pool.Close()

	os.Exit(exitCode)
}
