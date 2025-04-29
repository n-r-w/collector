package finalizer

import (
	"context"

	"github.com/n-r-w/collector/internal/entity"
)

//go:generate mockgen -source interface.go -destination interface_mock.go -package finalizer

// ICollectionReader is responsible for reading collection data.
type ICollectionReader interface {
	// GetCollections returns all active collections.
	GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error)
}

// IStatusChanger is responsible for changing the status of collection.
type IStatusChanger interface {
	UpdateStatus(ctx context.Context, collectionID entity.CollectionID, status entity.CollectionStatus) error
}

// IResultChanGetter is responsible for retrieving collection results.
type IResultChanGetter interface {
	GetResultChan(ctx context.Context, collectionID entity.CollectionID, limit int) (<-chan entity.RequestChunk, error)
}

// IResultChanSaver is responsible for saving collection results.
type IResultChanSaver interface {
	SaveResultChan(
		ctx context.Context, collectionID entity.CollectionID, requests <-chan entity.RequestChunk) (entity.ResultID, error)
}

// ICollectionResultUpdater is responsible for updating collection result ID.
type ICollectionResultUpdater interface {
	UpdateResultID(ctx context.Context, collectionID entity.CollectionID, resultID entity.ResultID) error
}

// ILocker is a service for locking resources.
type ILocker interface {
	// TryLockFunc executes a function with a lock.
	TryLockFunc(ctx context.Context, key entity.LockKey, fn func(context.Context) error) (acquired bool, err error)
}
