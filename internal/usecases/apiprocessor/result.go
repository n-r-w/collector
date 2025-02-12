package apiprocessor

import (
	"context"
	"fmt"

	"github.com/n-r-w/ammo-collector/internal/entity"
)

// GetResult returns a channel that receives result chunks.
func (s *Service) GetResult(ctx context.Context, collectionID entity.CollectionID) (<-chan entity.RequestChunk, error) {
	collection, err := s.collectionReader.GetCollection(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	if collection.Status != entity.StatusCompleted {
		return nil, fmt.Errorf("collection %d is not completed: %w", collectionID, entity.ErrInvalidStatus)
	}

	if collection.ResultID.IsAbsent() {
		resultChan := make(chan entity.RequestChunk)
		close(resultChan)
		return resultChan, nil
	}

	resultChan, err := s.resultGetter.GetResult(ctx, collection.ResultID.OrEmpty())
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	return resultChan, nil
}
