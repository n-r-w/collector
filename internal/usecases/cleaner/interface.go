package cleaner

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/entity"
)

// ILocker is a service for locking resources.
type ILocker interface {
	// TryLockFunc executes a function with a lock.
	TryLockFunc(ctx context.Context, key entity.LockKey, fn func(context.Context) error) (acquired bool, err error)
}

// ICleaner cleans up database.
type ICleaner interface {
	// Clean cleans up database.
	Clean(ctx context.Context, collectionIDs []entity.CollectionID) error
}

// ICollectionReader is responsible for reading collection data.
type ICollectionReader interface {
	// GetCollections returns all active collections.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
}
