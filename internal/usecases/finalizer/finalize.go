package finalizer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/ctxlog"
	"golang.org/x/sync/errgroup"
)

// finalizeCollections finalizes a list of collections.
func (s *Service) finalizeCollections(ctxMain context.Context, collections []entity.Collection) error {
	// use errgroup to easily limit concurrency
	// do not stop the execution when an error occurs, but accumulate errors
	var (
		errGroup     errgroup.Group
		errTotal     error
		successCount int
		muErrTotal   sync.Mutex
	)
	errGroup.SetLimit(s.cfg.Collection.FinalizerConcurrency)

	for i, collection := range collections {
		if i >= s.cfg.Collection.FinalizerMaxCollections {
			break
		}

		errGroup.Go(func() error {
			// try to get a lock by collection ID. This also opens the transaction.
			acquired, err := s.locker.TryLockFunc(ctxMain, entity.LockKey(collection.ID),
				func(ctxLock context.Context) error {
					ctxlog.Debug(ctxLock, "finalizer lock acquired",
						slog.String("collection_id", collection.ID.String()))

					if err := s.finalizeCollectionHelper(ctxLock, collection); err != nil {
						muErrTotal.Lock()
						errTotal = errors.Join(errTotal, err)
						muErrTotal.Unlock()
						return err // return error to rollback transaction
					} else {
						muErrTotal.Lock()
						successCount++
						muErrTotal.Unlock()
					}
					return nil
				})
			if !acquired && err != nil {
				// we did not reach the call to the function in TryLockFunc
				muErrTotal.Lock()
				errTotal = errors.Join(errTotal, fmt.Errorf("failed to get lock: %w", err))
				muErrTotal.Unlock()
				return nil
			}

			if acquired {
				ctxlog.Debug(ctxMain, "finalizer lock released",
					slog.String("collection_id", collection.ID.String()))
			} else {
				// lock is already acquired
				ctxlog.Debug(ctxMain, "finalizer lock is already acquired, skipping",
					slog.String("collection_id", collection.ID.String()))
			}

			return nil
		})
	}

	_ = errGroup.Wait()

	if errTotal != nil {
		if successCount == 0 {
			return fmt.Errorf("failed to finalize collections: %w", errTotal)
		}

		if !errors.Is(errTotal, context.Canceled) {
			ctxlog.Error(ctxMain, "part of collections failed to finalize",
				slog.Int("success_count", successCount),
				slog.Int("total_count", len(collections)),
				slog.Any("error", errTotal))
		}
	}

	ctxlog.Debug(ctxMain, "finalization completed",
		slog.Int("success_count", successCount),
		slog.Int("total_count", len(collections)))

	return nil
}

// finalizeCollectionsHelper orchestrates the process of finalizing collections.
func (s *Service) finalizeCollectionHelper(ctx context.Context, collection entity.Collection) error {
	ctxlog.Debug(ctx, "finalizing collection", slog.String("collection_id", collection.ID.String()))

	// if collection has no requests, skip result saving
	if collection.RequestCount > 0 {
		// Fetch requestsCh for this collection
		requestsCh, err := s.resultGetter.GetResultChan(ctx, collection.ID, collection.Task.Completion.RequestCountLimit)
		if err != nil {
			return fmt.Errorf("failed to fetch collection requests for collection %d: %w", collection.ID, err)
		}

		// save results
		// We are actually saving results in S3 inside a PostgreSQL transaction. This is bad.
		// But in this case, a long transaction is allowed, because:
		// 1) Only one finalizer can be running at the same time for one collection.
		// 2) Writing changes are possible only for incoming requests from Kafka.
		// 3) But they only add new records, not change existing ones.

		resultID, err := s.resultSaver.SaveResultChan(ctx, collection.ID, requestsCh)
		if err != nil {
			return fmt.Errorf("failed to save result for collection %d: %w", collection.ID, err)
		}

		// update collection result_id
		if err := s.resultUpdater.UpdateResultID(ctx, collection.ID, resultID); err != nil {
			return fmt.Errorf("failed to update collection result_id: %w", err)
		}
	}

	// In case of errors below, we will leave an "orphaned" archive of results in S3, but this
	// is not critical.
	if err := s.statusChanger.UpdateStatus(ctx, collection.ID, entity.StatusCompleted); err != nil {
		if errors.Is(err, entity.ErrCollectionNotFound) {
			// it's ok if someone else already finalized collection
			return nil
		}

		return fmt.Errorf("failed to update collections status: %w", err)
	}

	ctxlog.Debug(ctx, "collection finalized", slog.String("collection_id", collection.ID.String()))

	return nil
}
