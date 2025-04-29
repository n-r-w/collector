package cache

import (
	"context"

	"github.com/n-r-w/collector/internal/entity"
)

//go:generate mockgen -source interface.go -destination interface_mock.go -package cache

// ICollectionReader is responsible for reading collection data.
type ICollectionReader interface {
	// GetCollections returns all active collections.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
}
