package apiprocessor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/ctxlog"
)

// CreateCollection creates new collection with given criteria.
func (s *Service) CreateCollection(
	ctx context.Context, task entity.Task,
) (entity.CollectionID, error) {
	collectionID, err := s.collectionCreator.CreateCollection(ctx, task)
	if err != nil {
		ctxlog.Error(ctx, "failed to create collection",
			slog.Any("criteria", task),
			slog.Any("error", err))
		return 0, fmt.Errorf("create collection: %w", err)
	}

	ctxlog.Debug(ctx, "created new collection",
		slog.String("collection_id", collectionID.String()))
	return collectionID, nil
}
