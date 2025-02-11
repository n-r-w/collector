package finalizer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/bootstrap/executor"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// Service is responsible for finalizing data collection for active collections based on criteria.
type Service struct {
	txManager        txmgr.ITransactionManager
	collectionReader ICollectionReader
	statusChanger    IStatusChanger
	resultGetter     IResultChanGetter
	resultSaver      IResultChanSaver
	resultUpdater    ICollectionResultUpdater
	executor         *executor.Service
	locker           ILocker

	cfg *config.Config
}

// New creates a new Finalizer instance.
func New(
	cfg *config.Config,
	txManager txmgr.ITransactionManager,
	collectionReader ICollectionReader,
	statusChanger IStatusChanger,
	resultGetter IResultChanGetter,
	resultSaver IResultChanSaver,
	resultUpdater ICollectionResultUpdater,
	locker ILocker,
) (*Service, error) {
	s := &Service{
		txManager:        txManager,
		collectionReader: collectionReader,
		statusChanger:    statusChanger,
		resultGetter:     resultGetter,
		resultSaver:      resultSaver,
		resultUpdater:    resultUpdater,
		locker:           locker,
		cfg:              cfg,
	}

	var err error
	s.executor, err = executor.New("finalizer",
		&worker{service: s},
		cfg.Collection.FinalizerInterval,
		executor.WithJitter(cfg.Collection.FinalizerIntervalJitter),
		executor.WithOnError(func(ctx context.Context, err error) {
			ctxlog.Error(ctx, "finalizer error", slog.Any("error", err))
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
		Name: "Finalizer",
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
