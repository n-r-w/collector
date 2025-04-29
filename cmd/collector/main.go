package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/controller/handlers"
	"github.com/n-r-w/collector/internal/prepare"
	"github.com/n-r-w/collector/internal/prepare/di"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/grpcsrv"
	"github.com/n-r-w/pgh/v2/px/db"
)

func main() {
	cfg := config.MustNew()

	ctx := ctxlog.MustContext(context.Background(),
		ctxlog.WithName(cfg.App.ServiceName),
		ctxlog.WithLevel(cfg.App.LogLevel),
		ctxlog.WithEnvType(cfg.App.EnvType),
	)

	if err := run(ctx, cfg); err != nil {
		ctxlog.Error(ctx, "failed to run", slog.Any("error", err))
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context, cfg *config.Config) error {
	opts, err := getBootOptions(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare: %w", err)
	}

	opts = append(opts, bootstrap.WithLogger(ctxlog.FromContext(ctx)))

	b, err := bootstrap.New(cfg.App.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to create bootstrap: %w", err)
	}

	if err := b.Run(ctx, opts...); err != nil {
		return fmt.Errorf("failed to run bootstrap: %w", err)
	}

	return nil
}

func getBootOptions(ctx context.Context, cfg *config.Config) ([]bootstrap.Option, error) {
	var cleanUp []bootstrap.CleanUpFunc

	// Initialize tracing
	cleanupTracer, err := prepare.InitTracer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	if cleanupTracer != nil {
		cleanUp = append(cleanUp, cleanupTracer)
	}

	// Initialize metrics
	metrics, cleanupMetrics, err := prepare.InitMetrics(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}
	if cleanupMetrics != nil {
		cleanUp = append(cleanUp, cleanupMetrics)
	}

	// HTTP httpHandlers
	httpHandlers := handlers.NewHTTPHandlers()

	// Prepare GRPC server options
	grpcsrvOpts, err := grpcsrv.GetCtxLogOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get context logger options: %w", err)
	}
	grpcsrvOpts = append(grpcsrvOpts,
		grpcsrv.WithName(cfg.App.ServiceName),
		grpcsrv.WithEndpoint(grpcsrv.Endpoint{
			GRPC: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.GRPC.Port),
			HTTP: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.HTTP.Port),
		}),
		grpcsrv.WithHTTPReadHeaderTimeout(cfg.HTTP.ReadHeaderTimeout),
		grpcsrv.WithRegisterHTTPEndpoints(httpHandlers.GetHTTPEndpoints),
	)
	if cfg.Server.RecoverPanics {
		grpcsrvOpts = append(grpcsrvOpts, grpcsrv.WithRecover())
	}
	if cfg.Server.ProfilingEndpoint != "" {
		grpcsrvOpts = append(grpcsrvOpts, grpcsrv.WithPprof(cfg.Server.ProfilingEndpoint))
	}
	if cfg.Prometheus.Endpoint != "" {
		grpcsrvOpts = append(grpcsrvOpts, grpcsrv.WithMetrics(cfg.Prometheus.Endpoint))
	}

	// Initialize the DI container
	container, err := di.InitializeContainer(ctx, cfg,
		// Metrics interface
		metrics,
		// Database options
		[]db.Option{db.WithDSN(cfg.Database.DSN)},
		// GRPC server options
		grpcsrvOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}

	// Some manual injection of dependencies
	httpHandlers.SetResultGetter(container.APIProcessorService)

	// define boot order
	bootOpts := []bootstrap.Option{
		bootstrap.WithOrdered(container.Database, container.CacheService),
		bootstrap.WithUnordered(
			container.FinalizerService,
			container.APIProcessorService,
			container.RequestProcessorService,
			container.CleanupService,
			container.S3,
		),
		bootstrap.WithAfterStart(
			container.GRPCServer,
			container.KafkaConsumer),
		bootstrap.WithCleanUp(cleanUp...),
	}

	return bootOpts, nil
}
