package cleaner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/bootstrap/executor"
	"github.com/n-r-w/ctxlog"
)

// Service is responsible for cleaning up database.
type Service struct {
	cfg                  *config.Config
	executor             *executor.Service
	collectionReader     ICollectionReader
	locker               ILocker
	databaseCleaner      IDatabaseCleaner
	objectStorageCleaner IObjectStorageCleaner
}

// New creates new cleanup service.
func New(
	cfg *config.Config, locker ILocker, collectionReader ICollectionReader,
	databaseCleaner IDatabaseCleaner, objectStorageCleaner IObjectStorageCleaner,
) (*Service, error) {
	s := &Service{
		cfg:                  cfg,
		locker:               locker,
		collectionReader:     collectionReader,
		databaseCleaner:      databaseCleaner,
		objectStorageCleaner: objectStorageCleaner,
	}

	var err error
	s.executor, err = executor.New("cleaner",
		&worker{service: s},
		cfg.Collection.CleanupInterval,
		executor.WithJitter(cfg.Collection.CleanupIntervalJitter),
		executor.WithOnError(func(ctx context.Context, err error) {
			ctxlog.Error(ctx, "cleanup error", slog.Any("error", err))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("new executor: %w", err)
	}

	return s, nil
}

var _ bootstrap.IService = (*Service)(nil)

// Info returns service info. Implements bootstrap.IService Info method.
func (s *Service) Info() bootstrap.Info {
	return bootstrap.Info{
		Name: "Cleanup",
	}
}

// Start starts the service. Implements bootstrap.IService Start method.
func (s *Service) Start(ctx context.Context) error {
	if err := s.executor.Start(ctx); err != nil {
		return fmt.Errorf("start executor: %w", err)
	}

	return nil
}

// Stop stops the service. Implements bootstrap.IService Stop method.
func (s *Service) Stop(ctx context.Context) error {
	if err := s.executor.Stop(ctx); err != nil {
		return fmt.Errorf("stop executor: %w", err)
	}

	return nil
}

// worker is an implementation of executor.IExecutor.
type worker struct {
	service *Service
}

var _ executor.IExecutor = (*worker)(nil)

// Execute implements executor.Executor Execute method.
func (w *worker) Execute(ctx context.Context) error {
	return w.service.worker(ctx)
}

// StopExecutor implements executor.Executor StopExecutor method.
func (w *worker) StopExecutor(_ context.Context) error {
	return nil
}
