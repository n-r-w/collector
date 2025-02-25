package finalizer

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cfg := &config.Config{}
		cfg.Collection.FinalizerInterval = time.Second
		cfg.Collection.FinalizerIntervalJitter = time.Millisecond * 100

		svc, err := New(
			cfg,
			nil,
			NewMockICollectionReader(ctrl),
			NewMockIStatusChanger(ctrl),
			NewMockIResultChanGetter(ctrl),
			NewMockIResultChanSaver(ctrl),
			NewMockICollectionResultUpdater(ctrl),
			NewMockILocker(ctrl),
		)

		require.NoError(t, err)
		require.NotNil(t, svc)
	})
}

func TestService_finalizeCollections(t *testing.T) {
	ctx := ctxlog.MustContext(context.Background(),
		ctxlog.WithTesting(t),
		ctxlog.WithLevel(slog.LevelDebug),
	)

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cfg := &config.Config{}
		cfg.Collection.FinalizerConcurrency = 2
		cfg.Collection.FinalizerMaxCollections = 10

		mockLocker := NewMockILocker(ctrl)
		mockResultGetter := NewMockIResultChanGetter(ctrl)
		mockResultSaver := NewMockIResultChanSaver(ctrl)
		mockResultUpdater := NewMockICollectionResultUpdater(ctrl)
		mockStatusChanger := NewMockIStatusChanger(ctrl)

		svc := &Service{
			cfg:           cfg,
			locker:        mockLocker,
			resultGetter:  mockResultGetter,
			resultSaver:   mockResultSaver,
			resultUpdater: mockResultUpdater,
			statusChanger: mockStatusChanger,
		}

		collections := []entity.Collection{
			{
				ID:           entity.CollectionID(1),
				RequestCount: 100,
				Task: entity.Task{
					Completion: entity.CompletionCriteria{
						RequestCountLimit: 1000,
					},
				},
			},
		}

		resultChan := make(chan entity.RequestChunk)
		go func() {
			close(resultChan)
		}()

		mockLocker.EXPECT().
			TryLockFunc(gomock.Any(), entity.LockKey(1), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ entity.LockKey, fn func(context.Context) error) (bool, error) {
				return true, fn(ctx)
			})

		mockResultGetter.EXPECT().
			GetResultChan(gomock.Any(), entity.CollectionID(1), 1000).
			Return(resultChan, nil)

		mockResultSaver.EXPECT().
			SaveResultChan(gomock.Any(), entity.CollectionID(1), resultChan).
			Return(entity.ResultID("result-1"), nil)

		mockResultUpdater.EXPECT().
			UpdateResultID(gomock.Any(), entity.CollectionID(1), entity.ResultID("result-1")).
			Return(nil)

		mockStatusChanger.EXPECT().
			UpdateStatus(gomock.Any(), entity.CollectionID(1), entity.StatusCompleted).
			Return(nil)

		err := svc.finalizeCollections(ctx, collections)
		require.NoError(t, err)
	})

	t.Run("lock already acquired", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cfg := &config.Config{}
		cfg.Collection.FinalizerConcurrency = 2
		cfg.Collection.FinalizerMaxCollections = 10

		mockLocker := NewMockILocker(ctrl)

		svc := &Service{
			cfg:    cfg,
			locker: mockLocker,
		}

		collections := []entity.Collection{
			{
				ID: entity.CollectionID(1),
			},
		}

		mockLocker.EXPECT().
			TryLockFunc(gomock.Any(), entity.LockKey(1), gomock.Any()).
			Return(false, nil)

		err := svc.finalizeCollections(ctx, collections)
		require.NoError(t, err)
	})

	t.Run("lock error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cfg := &config.Config{}
		cfg.Collection.FinalizerConcurrency = 2
		cfg.Collection.FinalizerMaxCollections = 10

		mockLocker := NewMockILocker(ctrl)

		svc := &Service{
			cfg:    cfg,
			locker: mockLocker,
		}

		collections := []entity.Collection{
			{
				ID: entity.CollectionID(1),
			},
			{
				ID: entity.CollectionID(2),
			},
		}

		lockErr := errors.New("lock error")

		// Первая коллекция вызовет ошибку
		mockLocker.EXPECT().
			TryLockFunc(gomock.Any(), entity.LockKey(1), gomock.Any()).
			Return(false, lockErr)

		// Вторая коллекция тоже вызовет ошибку
		mockLocker.EXPECT().
			TryLockFunc(gomock.Any(), entity.LockKey(2), gomock.Any()).
			Return(false, lockErr)

		err := svc.finalizeCollections(ctx, collections)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to finalize collections")
		require.Contains(t, err.Error(), "failed to get lock")
	})
}

func TestService_worker(t *testing.T) {
	ctx := ctxlog.MustContext(context.Background(),
		ctxlog.WithTesting(t),
		ctxlog.WithLevel(slog.LevelDebug),
	)

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		cfg := &config.Config{}
		cfg.Collection.FinalizerConcurrency = 2
		cfg.Collection.FinalizerMaxCollections = 10

		mockReader := NewMockICollectionReader(ctrl)
		mockLocker := NewMockILocker(ctrl)
		mockStatusChanger := NewMockIStatusChanger(ctrl)

		svc := &Service{
			cfg:              cfg,
			collectionReader: mockReader,
			locker:           mockLocker,
			statusChanger:    mockStatusChanger,
		}

		collections := []entity.Collection{
			{
				ID: entity.CollectionID(1),
				Task: entity.Task{
					Completion: entity.CompletionCriteria{
						TimeLimit: time.Minute,
					},
				},
			},
		}

		mockReader.EXPECT().
			GetCollections(gomock.Any(), entity.CollectionFilter{
				Statuses: entity.ActiveCollectionStatuses(),
			}).
			Return(collections, nil)

		mockLocker.EXPECT().
			TryLockFunc(gomock.Any(), entity.LockKey(1), gomock.Any()).
			Return(false, nil)

		err := svc.worker(ctx)
		require.NoError(t, err)
	})

	t.Run("no collections", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockReader := NewMockICollectionReader(ctrl)

		svc := &Service{
			collectionReader: mockReader,
		}

		mockReader.EXPECT().
			GetCollections(gomock.Any(), entity.CollectionFilter{
				Statuses: entity.ActiveCollectionStatuses(),
			}).
			Return(nil, nil)

		err := svc.worker(ctx)
		require.NoError(t, err)
	})

	t.Run("get collections error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockReader := NewMockICollectionReader(ctrl)

		svc := &Service{
			collectionReader: mockReader,
		}

		mockReader.EXPECT().
			GetCollections(gomock.Any(), entity.CollectionFilter{
				Statuses: entity.ActiveCollectionStatuses(),
			}).
			Return(nil, errors.New("db error"))

		err := svc.worker(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "get collections")
	})
}
