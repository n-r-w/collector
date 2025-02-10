package apiprocessor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
)

// GetCollections returns collections by filter.
func (s *Service) GetCollections(
	ctx context.Context, filter entity.CollectionFilter,
) ([]entity.Collection, error) {
	collections, err := s.collectionReader.GetCollections(ctx, filter)
	if err != nil {
		ctxlog.Error(ctx, "failed to get collections",
			slog.Any("filter", filter),
			slog.Any("error", err))
		return nil, fmt.Errorf("get collections: %w", err)
	}

	ctxlog.Debug(ctx, "retrieved collections", slog.Int("count", len(collections)))
	return collections, nil
}

// GetCollection returns the specific collection.
func (s *Service) GetCollection(
	ctx context.Context, collectionID entity.CollectionID,
) (entity.Collection, error) {
	collection, err := s.collectionReader.GetCollection(ctx, collectionID)
	if err != nil {
		ctxlog.Error(ctx, "failed to get collection",
			slog.String("collection_id", collectionID.String()),
			slog.Any("error", err))
		return entity.Collection{}, fmt.Errorf("get collection: %w", err)
	}

	ctxlog.Debug(ctx, "retrieved collection", slog.String("collection_id", collectionID.String()))
	return collection, nil
}
