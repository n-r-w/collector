package cleaner

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWorker(t *testing.T) {
	ctx := ctxlog.MustContext(context.Background(),
		ctxlog.WithTesting(t),
		ctxlog.WithLevel(slog.LevelInfo),
	)

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockLocker := NewMockILocker(ctrl)
		mockDB := NewMockIDatabaseCleaner(ctrl)
		mockOS := NewMockIObjectStorageCleaner(ctrl)
		mockReader := NewMockICollectionReader(ctrl)

		now := time.Now()

		cfg := &config.Config{}
		cfg.Collection.RetentionPeriod = time.Hour * 24 * 7

		svc := &Service{
			locker:               mockLocker,
			databaseCleaner:      mockDB,
			objectStorageCleaner: mockOS,
			collectionReader:     mockReader,
			cfg:                  cfg,
			now:                  func() time.Time { return now },
		}

		collections := []entity.Collection{{
			ID:        entity.CollectionID(1),
			CreatedAt: now.Add(-time.Hour * 24 * 8), // 8 days old
			ResultID:  mo.Some(entity.ResultID("result-id")),
		}}

		mockReader.EXPECT().GetCollections(gomock.Any(),
			entity.CollectionFilter{
				ToTime: mo.Some(now.Add(-time.Hour * 24 * 7)),
			},
		).Return(collections, nil)

		mockLocker.EXPECT().TryLockFunc(gomock.Any(), entity.CleanUpLockKey, gomock.Any()).
			DoAndReturn(func(ctx context.Context, key entity.LockKey, fn func(context.Context) error) (bool, error) {
				return true, fn(ctx)
			})

		mockDB.EXPECT().CleanDatabase(gomock.Any(), []entity.CollectionID{entity.CollectionID(1)}).Return(nil)
		mockOS.EXPECT().CleanObjectStorage(gomock.Any(), []entity.ResultID{entity.ResultID("result-id")}).Return(nil)

		err := svc.worker(ctx)
		require.NoError(t, err)
	})

	t.Run("lock failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockLocker := NewMockILocker(ctrl)
		mockReader := NewMockICollectionReader(ctrl)
		mockDB := NewMockIDatabaseCleaner(ctrl)
		mockOS := NewMockIObjectStorageCleaner(ctrl)

		now := time.Now()

		cfg := &config.Config{}
		cfg.Collection.RetentionPeriod = time.Hour * 24 * 7

		svc := &Service{
			locker:               mockLocker,
			collectionReader:     mockReader,
			databaseCleaner:      mockDB,
			objectStorageCleaner: mockOS,
			cfg:                  cfg,
			now:                  func() time.Time { return now },
		}

		collections := []entity.Collection{{
			ID:        entity.CollectionID(1),
			CreatedAt: now.Add(-time.Hour * 24 * 8), // 8 days old
		}}

		mockReader.EXPECT().GetCollections(gomock.Any(),
			entity.CollectionFilter{
				ToTime: mo.Some(now.Add(-time.Hour * 24 * 7)),
			},
		).Return(collections, nil)

		mockLocker.EXPECT().TryLockFunc(gomock.Any(), entity.CleanUpLockKey, gomock.Any()).
			Return(false, nil)

		err := svc.worker(ctx)
		require.NoError(t, err)
	})
}
