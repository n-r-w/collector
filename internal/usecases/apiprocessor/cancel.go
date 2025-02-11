package apiprocessor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// CancelCollection stops an active collection.
func (s *Service) CancelCollection(ctx context.Context, collectionID entity.CollectionID) error {
	return s.trManager.Begin(ctx, func(ctx context.Context) error {
		return s.cancelCollectionHelper(ctx, collectionID)
	}, txmgr.WithLock())
}

func (s *Service) cancelCollectionHelper(ctx context.Context, collectionID entity.CollectionID) error {
	// Get current status (with lock record)
	collection, err := s.collectionReader.GetCollection(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("get collection: %w", err)
	}

	// Only allow stopping in-progress collections
	if collection.Status != entity.StatusInProgress && collection.Status != entity.StatusPending {
		return errors.New("collection is not in active state")
	}

	// Mark as stopped
	if err := s.collectionUpdater.UpdateStatus(ctx, collectionID, entity.StatusCancelled); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	ctxlog.Debug(ctx, "collection cancelled",
		slog.String("collection_id", collectionID.String()))
	return nil
}
