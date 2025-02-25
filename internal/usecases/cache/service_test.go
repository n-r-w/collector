package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Test fixtures

func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Collection.CacheUpdateInterval = time.Second
	cfg.Collection.CacheUpdateIntervalJitter = time.Millisecond * 100
	return cfg
}

var testCollections = []entity.Collection{
	{
		ID:     entity.CollectionID(1),
		Status: entity.StatusInProgress,
	},
	{
		ID:     entity.CollectionID(2),
		Status: entity.StatusInProgress,
	},
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		setupReader func(ctrl *gomock.Controller) ICollectionReader
		wantErr     error
	}{
		{
			name: "success",
			cfg:  testConfig(),
			setupReader: func(ctrl *gomock.Controller) ICollectionReader {
				return NewMockICollectionReader(ctrl)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reader := tt.setupReader(ctrl)
			svc, err := New(tt.cfg, reader)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, svc)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, svc)
				assert.NotNil(t, svc.collections)
				assert.NotNil(t, svc.executor)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	tests := []struct {
		name        string
		collections []entity.Collection
		setup       func(*Service)
	}{
		{
			name:        "empty cache",
			collections: []entity.Collection{},
			setup:       nil,
		},
		{
			name:        "populated cache",
			collections: testCollections,
			setup: func(s *Service) {
				s.update(testCollections)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc, err := New(testConfig(), NewMockICollectionReader(ctrl))
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(svc)
			}

			got := svc.Get()
			assert.Equal(t, len(tt.collections), len(got))

			// Verify collections match
			for _, c := range tt.collections {
				found := false
				for _, g := range got {
					if g.ID == c.ID {
						assert.Equal(t, c, g)
						found = true
						break
					}
				}
				assert.True(t, found, "Collection %d not found", c.ID)
			}
		})
	}
}

func TestService_worker(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockICollectionReader)
		expectedError bool
		checkCache    func(*testing.T, *Service)
	}{
		{
			name: "successful update",
			setupMock: func(m *MockICollectionReader) {
				m.EXPECT().
					GetCollections(gomock.Any(), entity.CollectionFilter{
						Statuses: entity.CollectingCollectionStatuses(),
					}).
					Return(testCollections, nil)
			},
			expectedError: false,
			checkCache: func(t *testing.T, s *Service) {
				got := s.Get()
				assert.Equal(t, len(testCollections), len(got))
			},
		},
		{
			name: "error getting collections",
			setupMock: func(m *MockICollectionReader) {
				m.EXPECT().
					GetCollections(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("db error"))
			},
			expectedError: true,
			checkCache: func(t *testing.T, s *Service) {
				got := s.Get()
				assert.Empty(t, got)
			},
		},
		{
			name: "context canceled",
			setupMock: func(m *MockICollectionReader) {
				m.EXPECT().
					GetCollections(gomock.Any(), gomock.Any()).
					Return(nil, context.Canceled)
			},
			expectedError: false,
			checkCache: func(t *testing.T, s *Service) {
				got := s.Get()
				assert.Empty(t, got)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reader := NewMockICollectionReader(ctrl)
			tt.setupMock(reader)

			svc, err := New(testConfig(), reader)
			require.NoError(t, err)

			ctx := context.Background()
			err = svc.worker(ctx)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.checkCache(t, svc)
		})
	}
}

func TestService_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reader := NewMockICollectionReader(ctrl)
	reader.EXPECT().
		GetCollections(gomock.Any(), gomock.Any()).
		Return(testCollections, nil).
		AnyTimes()

	svc, err := New(testConfig(), reader)
	require.NoError(t, err)

	ctx := context.Background()

	// Test Start
	err = svc.Start(ctx)
	require.NoError(t, err)

	// Wait for at least one update
	time.Sleep(testConfig().Collection.CacheUpdateInterval + testConfig().Collection.CacheUpdateIntervalJitter)

	// Verify cache was updated
	got := svc.Get()
	assert.NotEmpty(t, got)

	// Test Stop
	err = svc.Stop(ctx)
	assert.NoError(t, err)
}

func TestService_Concurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reader := NewMockICollectionReader(ctrl)
	svc, err := New(testConfig(), reader)
	require.NoError(t, err)

	// Setup initial data
	svc.update(testCollections)

	// Test concurrent reads
	const numReaders = 10
	done := make(chan bool)

	for range numReaders {
		go func() {
			for range 100 {
				collections := svc.Get()
				assert.Equal(t, len(testCollections), len(collections))
			}
			done <- true
		}()
	}

	// Wait for all readers
	for range numReaders {
		<-done
	}

	// Test concurrent read/write
	go func() {
		for range 100 {
			svc.update(testCollections)
		}
		done <- true
	}()

	go func() {
		for range 100 {
			_ = svc.Get()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	collections := svc.Get()
	assert.Equal(t, len(testCollections), len(collections))
}
