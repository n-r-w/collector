package finalizer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/n-r-w/bootstrap/executor"
	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/ctxlog"
)

// worker retrieves a list of active collections and checks if they need finalization.
func (s *Service) worker(ctx context.Context) error {
	collections, err := s.collectionReader.GetCollections(ctx, entity.CollectionFilter{
		Statuses: entity.ActiveCollectionStatuses(),
	})
	if err != nil {
		return fmt.Errorf("get collections: %w", err)
	}

	if len(collections) == 0 {
		return nil
	}

	var toFinalize []entity.Collection
	for _, collection := range collections {
		if collection.IsOutOfTimeLimit() || collection.IsOutOfRequestLimit() {
			toFinalize = append(toFinalize, collection)
		}
	}

	if len(toFinalize) == 0 {
		return nil
	}

	if err := s.finalizeCollections(ctx, toFinalize); err != nil {
		if !errors.Is(err, context.Canceled) {
			ctxlog.Error(ctx, "finalize collections error", slog.Any("error", err))
		}
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
