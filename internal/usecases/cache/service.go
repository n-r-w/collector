package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/bootstrap/executor"
	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/collector/internal/usecases/reqprocessor"
	"github.com/n-r-w/ctxlog"
)

// Service implements collection getter.
type Service struct {
	collections      map[entity.CollectionID]entity.Collection
	mu               sync.RWMutex
	collectionReader ICollectionReader
	executor         *executor.Service
}

var (
	_ bootstrap.IService             = (*Service)(nil)
	_ reqprocessor.ICollectionCacher = (*Service)(nil)
)

// New creates new collection cache service.
func New(
	cfg *config.Config,
	collectionGetter ICollectionReader,
) (*Service, error) {
	c := &Service{
		collections:      make(map[entity.CollectionID]entity.Collection),
		collectionReader: collectionGetter,
	}

	var err error
	c.executor, err = executor.New("collection_cache",
		&worker{service: c},
		cfg.Collection.CacheUpdateInterval,
		executor.WithJitter(cfg.Collection.CacheUpdateIntervalJitter),
		executor.WithOnError(func(ctx context.Context, err error) {
			ctxlog.Error(ctx, "collection_cache error", slog.Any("error", err))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("new executor: %w", err)
	}

	return c, nil
}

// Start begins the collection cache update process.
func (s *Service) Start(ctx context.Context) error {
	if err := s.executor.Start(ctx); err != nil {
		return fmt.Errorf("start executor: %w", err)
	}

	return nil
}

// Stop stops the collection cache update process.
func (s *Service) Stop(ctx context.Context) error {
	if err := s.executor.Stop(ctx); err != nil {
		return fmt.Errorf("stop executor: %w", err)
	}

	return nil
}

// Info returns the service info.
func (s *Service) Info() bootstrap.Info {
	return bootstrap.Info{
		Name: "Collection Cache",
	}
}

// Get returns collections from the cache.
func (s *Service) Get() []entity.Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collections := make([]entity.Collection, 0, len(s.collections))
	for _, collection := range s.collections {
		collections = append(collections, collection)
	}

	return collections
}

// update updates the cache with the provided collections.
func (s *Service) update(collections []entity.Collection) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.collections)

	for _, collection := range collections {
		s.collections[collection.ID] = collection
	}
}

// worker retrieves a list of active collections and updates the cache.
func (s *Service) worker(ctx context.Context) error {
	collections, err := s.collectionReader.GetCollections(ctx, entity.CollectionFilter{
		Statuses: entity.CollectingCollectionStatuses(),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return fmt.Errorf("get active collections: %w", err)
	}

	s.update(collections)

	return nil
}

// worker is a wrapper for executor.Executor.
type worker struct {
	service *Service
}

var _ executor.IExecutor = (*worker)(nil)

// Execute implements executor.Executor Execute method.
func (w *worker) Execute(ctx context.Context) error {
	return w.service.worker(ctx)
}

// StopExecutor implements executor.IExecutor StopExecutor method.
func (w *worker) StopExecutor(_ context.Context) error {
	return nil
}
