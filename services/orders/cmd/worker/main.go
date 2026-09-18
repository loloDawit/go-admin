// The worker runs the publisher and the projector in one process so the API
// stays stateless and can be restarted independently of either.
//
// One instance only: the publisher's ordering guarantee assumes a single sender
// draining the outbox by id.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/loloDawit/go-admin/platform/observability"
	pgxplatform "github.com/loloDawit/go-admin/platform/pgx"
	"github.com/loloDawit/go-admin/services/orders/internal/config"
	"github.com/loloDawit/go-admin/services/orders/internal/natsx"
	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
	"github.com/loloDawit/go-admin/services/orders/internal/reporting"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.ServiceName+"-worker", os.Stdout)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.NewTracerProvider(ctx, cfg.ServiceName+"-worker", cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("tracing", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	shutdownMetrics, err := observability.NewMeterProvider(ctx, cfg.ServiceName+"-worker", cfg.OTLPEndpoint)
	if err != nil {
		logger.Error("metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = shutdownMetrics(context.Background()) }()

	pool, err := pgxplatform.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	conn, js, err := natsx.Connect(ctx, cfg.NATSURL)
	if err != nil {
		logger.Error("broker", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	stream, err := natsx.EnsureStream(ctx, js, cfg.NATSDuplicateWindow)
	if err != nil {
		logger.Error("stream", slog.String("error", err.Error()))
		os.Exit(1)
	}
	consumer, err := natsx.EnsureConsumer(ctx, stream)
	if err != nil {
		logger.Error("consumer", slog.String("error", err.Error()))
		os.Exit(1)
	}

	outboxRepo := outbox.NewPostgresRepository(pool)
	outboxMetrics, err := outbox.NewMetrics()
	if err != nil {
		logger.Error("metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}
	publisher := outbox.NewPublisher(
		outboxRepo, js,
		cfg.OutboxBatchSize, cfg.OutboxPollInterval, logger, outboxMetrics,
	)
	projector := reporting.NewProjector(reporting.NewPostgresRepository(pool), consumer, logger)

	done := make(chan struct{}, 2)
	go func() {
		if err := publisher.Run(ctx); err != nil {
			logger.Error("publisher", slog.String("error", err.Error()))
		}
		done <- struct{}{}
	}()
	go func() {
		if err := projector.Run(ctx); err != nil {
			logger.Error("projector", slog.String("error", err.Error()))
		}
		done <- struct{}{}
	}()

	logger.Info("worker started",
		slog.Int("outbox_batch_size", cfg.OutboxBatchSize),
		slog.Duration("outbox_poll_interval", cfg.OutboxPollInterval),
	)

	<-ctx.Done()
	<-done
	<-done
}
