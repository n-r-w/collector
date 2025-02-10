package reqprocessor

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/entity"
)

// IRequestStorer is responsible for storing requests.
type IRequestStorer interface {
	Store(ctx context.Context, requests []entity.RequestContent, toStore []entity.MatchResult) error
}

// ICollectionCacher is responsible for manage active collections cache.
type ICollectionCacher interface {
	Get() []entity.Collection
}
