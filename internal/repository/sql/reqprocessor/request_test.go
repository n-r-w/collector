package reqprocessor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/repository/sql"
	"github.com/n-r-w/ammo-collector/internal/repository/sql/dbmodel"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, txmgr *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db, txmgr)
		},
	)

	// Create test collections
	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler: "test-handler",
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         time.Hour,
			RequestCountLimit: 2,
		},
	}

	collection1ID := sql.CreateTestCollection(t, ctx, s.conn, task)
	collection2ID := sql.CreateTestCollection(t, ctx, s.conn, task)

	// Prepare test data
	now := time.Now().UTC()
	headers := map[string][]string{
		"Content-Type": {"application/json"},
		"User-Agent":   {"test-agent"},
	}

	requests := []entity.RequestContent{
		{
			Handler:   "test-handler-1",
			Headers:   headers,
			Body:      []byte(`{"test": "data1"}`),
			CreatedAt: now,
		},
		{
			Handler:   "test-handler-2",
			Headers:   headers,
			Body:      []byte(`{"test": "data2"}`),
			CreatedAt: now,
		},
	}

	toStore := []entity.MatchResult{
		{
			RequestPos:    0,
			CollectionIDs: []entity.CollectionID{collection1ID},
		},
		{
			RequestPos:    1,
			CollectionIDs: []entity.CollectionID{collection1ID, collection2ID},
		},
	}

	// Store requests
	err := s.Store(ctx, requests, toStore)
	require.NoError(t, err)

	// Verify stored requests
	var storedRequests []dbmodel.Request
	err = px.Select(ctx, s.conn(ctx),
		pgh.Builder().Select("id", "handler", "headers", "body", "created_at").
			From("requests").
			OrderBy("id"), &storedRequests)
	require.NoError(t, err)
	require.Len(t, storedRequests, 2)

	// Verify first request
	headersJSON, err := json.Marshal(headers)
	require.NoError(t, err)

	require.Equal(t, "test-handler-1", storedRequests[0].Handler)
	require.JSONEq(t, string(headersJSON), string(storedRequests[0].Headers))
	require.JSONEq(t, `{"test": "data1"}`, string(storedRequests[0].Body))
	require.WithinDuration(t, now, storedRequests[0].CreatedAt, time.Second)

	// Verify second request
	require.Equal(t, "test-handler-2", storedRequests[1].Handler)
	require.JSONEq(t, string(headersJSON), string(storedRequests[1].Headers))
	require.JSONEq(t, `{"test": "data2"}`, string(storedRequests[1].Body))
	require.WithinDuration(t, now, storedRequests[1].CreatedAt, time.Second)

	// Verify request-collection links
	var links []dbmodel.RequestCollection
	err = px.Select(ctx, s.conn(ctx),
		pgh.Builder().Select("request_id", "collection_id").
			From("request_collections").
			OrderBy("request_id", "collection_id"), &links)
	require.NoError(t, err)
	require.Len(t, links, 3) // First request -> coll1, Second request -> coll1 + coll2
	require.EqualValues(t, 1, links[0].RequestID)
	require.EqualValues(t, 1, links[0].CollectionID)
	require.EqualValues(t, 2, links[1].RequestID)
	require.EqualValues(t, 1, links[1].CollectionID)
	require.EqualValues(t, 2, links[2].RequestID)
	require.EqualValues(t, 2, links[2].CollectionID)

	// Verify collection updates
	collection1, err := dbmodel.CollectionByID(ctx, s.conn(ctx), int64(collection1ID))
	require.NoError(t, err)
	require.EqualValues(t, 2, collection1.RequestCount) // Both requests linked
	require.True(t, collection1.StartedAt.Valid)
	require.True(t, collection1.UpdatedAt.Valid)
	require.EqualValues(t, entity.StatusFinalizing, collection1.Status)

	collection2, err := dbmodel.CollectionByID(ctx, s.conn(ctx), int64(collection2ID))
	require.NoError(t, err)
	require.EqualValues(t, 1, collection2.RequestCount) // Only second request linked
	require.True(t, collection2.StartedAt.Valid)
	require.True(t, collection2.UpdatedAt.Valid)
}
