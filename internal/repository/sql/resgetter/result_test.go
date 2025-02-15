package resgetter

import (
	"testing"
	"time"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/repository/sql"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/stretchr/testify/require"
)

func TestGetResultChan(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, txmgr *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db, txmgr)
		},
	)

	// Configure small batch size for testing
	s.cfg.Collection.FinalizerResultBatchSize = 2

	// Create test collection
	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler: "test-handler",
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         time.Hour,
			RequestCountLimit: 5,
		},
	}

	collectionID := sql.CreateTestCollection(t, ctx, s.conn, task)

	// Prepare test requests
	testData := [][]byte{
		[]byte(`{"test": "data1"}`),
		[]byte(`{"test": "data2"}`),
		[]byte(`{"test": "data3"}`),
		[]byte(`{"test": "data4"}`),
		[]byte(`{"test": "data5"}`),
	}

	// Insert test requests and link them to the collection
	for _, data := range testData {
		var requestID int64
		require.NoError(t, px.SelectOne(ctx, s.conn(ctx),
			pgh.Builder().
				Insert("requests").
				Columns("handler", "headers", "body", "created_at").
				Values("test-handler", []byte(`{}`), data, time.Now()).
				Suffix("RETURNING id"), &requestID))

		_, err := px.Exec(ctx, s.conn(ctx),
			pgh.Builder().
				Insert("request_collections").
				Columns("collection_id", "request_id").
				Values(collectionID, requestID))
		require.NoError(t, err)
	}

	// Test cases
	tests := []struct {
		name  string
		limit int
		want  int // expected number of results
	}{
		{"get all results", 5, 5},
		{"get partial results", 3, 3},
		{"get zero results", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Get results channel
			resultChan, err := s.GetResultChan(ctx, collectionID, tt.limit)
			require.NoError(t, err)

			// Collect results
			var results []entity.RequestChunk
			for result := range resultChan {
				require.NoError(t, result.Err)
				results = append(results, result)
			}

			// Verify results
			require.Len(t, results, tt.want)

			// Verify data content for received results
			for i, result := range results {
				require.Equal(t, testData[i], result.Data)
				require.JSONEq(t, string(testData[i]), string(result.Data))
			}
		})
	}
}
