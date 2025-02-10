package apiprocessor

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/entity"
)

// ICollectionCreator is responsible for creating new collections.
type ICollectionCreator interface {
	// CreateCollection creates a new collection with the given parameters and returns its ID.
	CreateCollection(ctx context.Context, task entity.Task) (entity.CollectionID, error)
}

// ICollectionReader is responsible for reading collection data.
type ICollectionReader interface {
	// GetCollections returns all active collections.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
	// GetCollection returns the status of a specific collection.
	GetCollection(ctx context.Context, id entity.CollectionID) (entity.Collection, error)
}

// ICollectionUpdater is responsible for updating collection status.
type ICollectionUpdater interface {
	// UpdateStatus updates collection status.
	UpdateStatus(ctx context.Context, collectionID entity.CollectionID, status entity.CollectionStatus) error
}

// IResultGetter is responsible for retrieving collection results.
type IResultGetter interface {
	// GetResult returns the result of a collection.
	GetResult(ctx context.Context, resultID entity.ResultID) (<-chan entity.RequestChunk, error)
}
