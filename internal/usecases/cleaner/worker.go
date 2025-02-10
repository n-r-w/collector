package cleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/moznion/go-optional"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
	"github.com/samber/lo"
)

// worker cleans up database.
func (s *Service) worker(ctx context.Context) error {
	ctxlog.Debug(ctx, "starting to clean up collections")

	// get all collections
	collections, err := s.collectionReader.GetCollections(ctx, entity.CollectionFilter{
		ToTime: optional.Some(time.Now().Add(-s.cfg.Collection.RetentionPeriod)),
	})
	if err != nil {
		return fmt.Errorf("get collections: %w", err)
	}

	if len(collections) == 0 {
		ctxlog.Debug(ctx, "no collections to clean up")
		return nil
	}

	// try to get a lock
	acquired, err := s.locker.TryLockFunc(ctx, entity.CleanUpLockKey,
		func(ctxLock context.Context) error {
			toCleanup := lo.Map(collections, func(c entity.Collection, index int) entity.CollectionID {
				return c.ID
			})

			// cleanup
			if errCleanup := s.cleaner.Clean(ctxLock, toCleanup); errCleanup != nil {
				return fmt.Errorf("clean collections: %w", errCleanup)
			}

			return nil
		})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		ctxlog.Error(ctx, "failed to get cleanup lock", slog.Any("error", err))

		return err
	}

	if !acquired {
		ctxlog.Debug(ctx, "cleanup lock is already acquired, skipping")
	} else {
		ctxlog.Debug(ctx, "finished cleaning up collections", slog.Int("count", len(collections)))
	}

	return nil
}
