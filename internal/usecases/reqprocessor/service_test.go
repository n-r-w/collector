package reqprocessor

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/ctxlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_HandleRequest(t *testing.T) {
	ctx := ctxlog.MustContext(context.Background(),
		ctxlog.WithTesting(t),
		ctxlog.WithLevel(slog.LevelDebug),
	)

	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) (*Service, []entity.RequestContent)
		wantErr   bool
		errString string
	}{
		{
			name: "empty collections",
			setup: func(t *testing.T) (*Service, []entity.RequestContent) {
				ctrl := gomock.NewController(t)

				requestStorer := NewMockIRequestStorer(ctrl)
				cacheGetter := NewMockICollectionCacher(ctrl)

				cacheGetter.EXPECT().Get().Return(nil)

				svc := New(requestStorer, cacheGetter)
				return svc, []entity.RequestContent{{Handler: "test"}}
			},
			wantErr: false,
		},
		{
			name: "no matching collections",
			setup: func(t *testing.T) (*Service, []entity.RequestContent) {
				ctrl := gomock.NewController(t)

				requestStorer := NewMockIRequestStorer(ctrl)
				cacheGetter := NewMockICollectionCacher(ctrl)

				cacheGetter.EXPECT().Get().Return([]entity.Collection{
					{
						ID:     1,
						Status: entity.StatusPending,
						Task: entity.Task{
							MessageSelection: entity.MessageSelectionCriteria{
								Handler: "other",
							},
						},
					},
				})

				svc := New(requestStorer, cacheGetter)
				return svc, []entity.RequestContent{{Handler: "test"}}
			},
			wantErr: false,
		},
		{
			name: "successful match and store",
			setup: func(t *testing.T) (*Service, []entity.RequestContent) {
				ctrl := gomock.NewController(t)

				requestStorer := NewMockIRequestStorer(ctrl)
				cacheGetter := NewMockICollectionCacher(ctrl)

				pattern := regexp.MustCompile("value.*")
				collections := []entity.Collection{
					{
						ID:     1,
						Status: entity.StatusPending,
						Task: entity.Task{
							MessageSelection: entity.MessageSelectionCriteria{
								Handler: "test",
								HeaderCriteria: []entity.HeaderCriteria{
									{
										HeaderName: "test-header",
										Pattern:    pattern,
									},
								},
							},
						},
					},
				}

				requests := []entity.RequestContent{
					{
						Handler: "test",
						Headers: map[string][]string{
							"test-header": {"value1"},
						},
					},
				}

				expectedMatches := []entity.MatchResult{
					{
						RequestPos:    0,
						CollectionIDs: []entity.CollectionID{1},
					},
				}

				cacheGetter.EXPECT().Get().Return(collections)
				requestStorer.EXPECT().
					Store(gomock.Any(), requests, expectedMatches).
					Return(nil)

				svc := New(requestStorer, cacheGetter)
				return svc, requests
			},
			wantErr: false,
		},
		{
			name: "store error",
			setup: func(t *testing.T) (*Service, []entity.RequestContent) {
				ctrl := gomock.NewController(t)

				requestStorer := NewMockIRequestStorer(ctrl)
				cacheGetter := NewMockICollectionCacher(ctrl)

				collections := []entity.Collection{
					{
						ID:     1,
						Status: entity.StatusPending,
						Task: entity.Task{
							MessageSelection: entity.MessageSelectionCriteria{
								Handler: "test",
							},
						},
					},
				}

				requests := []entity.RequestContent{{Handler: "test"}}
				expectedMatches := []entity.MatchResult{
					{
						RequestPos:    0,
						CollectionIDs: []entity.CollectionID{1},
					},
				}

				cacheGetter.EXPECT().Get().Return(collections)
				requestStorer.EXPECT().
					Store(gomock.Any(), requests, expectedMatches).
					Return(errors.New("store error"))

				svc := New(requestStorer, cacheGetter)
				return svc, requests
			},
			wantErr:   true,
			errString: "failed to store request: store error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, requests := tt.setup(t)
			err := svc.HandleRequest(ctx, requests)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
				return
			}

			require.NoError(t, err)
		})
	}
}
