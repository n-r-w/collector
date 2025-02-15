package cleaner

import (
	"context"

	"github.com/n-r-w/ammo-collector/internal/entity"
)

//go:generate mockgen -source interface.go -destination interface_mock.go -package cleaner

// ILocker is a service for locking resources.
type ILocker interface {
	// TryLockFunc executes a function with a lock.
	TryLockFunc(ctx context.Context, key entity.LockKey, fn func(context.Context) error) (acquired bool, err error)
}

// IDatabaseCleaner cleans up database.
type IDatabaseCleaner interface {
	// Clean cleans up database.
	CleanDatabase(ctx context.Context, collectionIDs []entity.CollectionID) error
}

// IObjectStorageCleaner cleans up object storage.
type IObjectStorageCleaner interface {
	// Clean cleans up object storage.
	CleanObjectStorage(ctx context.Context, resultIDs []entity.ResultID) error
}

// ICollectionReader is responsible for reading collection data.
type ICollectionReader interface {
	// GetCollections returns all active collections.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
}
