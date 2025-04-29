package handlers

import (
	"context"

	"github.com/n-r-w/collector/internal/entity"
)

// ICollectionManager is responsible for managing collections.
type ICollectionManager interface {
	// Collection creates a new collection with the given parameters and returns its ID.
	CreateCollection(ctx context.Context, task entity.Task) (entity.CollectionID, error)
	// GetCollections returns collections by filter.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
	// GetCollection returns the status of a specific collection.
	GetCollection(ctx context.Context, id entity.CollectionID) (entity.Collection, error)
	// CancelCollection terminates an active collection.
	CancelCollection(ctx context.Context, id entity.CollectionID) error
}

// IResultGetter is responsible for retrieving collection results by chunks.
type IResultGetter interface {
	GetResult(ctx context.Context, collectionID entity.CollectionID) (<-chan entity.RequestChunk, error)
}
