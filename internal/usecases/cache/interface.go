package cache

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/entity"
)

// ICollectionReader is responsible for reading collection data.
type ICollectionReader interface {
	// GetCollections returns all active collections.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
}
