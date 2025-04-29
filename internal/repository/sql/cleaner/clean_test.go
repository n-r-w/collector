package cleaner

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/collector/internal/repository/sql"
	"github.com/n-r-w/collector/internal/repository/sql/dbmodel"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/txmgr"
	sq "github.com/n-r-w/squirrel"
	"github.com/stretchr/testify/require"
)

func TestClean(t *testing.T) {
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
			RequestCountLimit: 100,
		},
	}

	collection1ID := sql.CreateTestCollection(t, ctx, s.conn, task)
	collection2ID := sql.CreateTestCollection(t, ctx, s.conn, task)

	// Create test requests and link them to collections
	requests := []struct {
		handler   string
		headers   map[string][]string
		body      []byte
		linkTo    []entity.CollectionID
		requestID int64
	}{
		{
			handler: "handler1",
			headers: map[string][]string{"Content-Type": {"application/json"}},
			body:    []byte(`{"test": "data1"}`),
			linkTo:  []entity.CollectionID{collection1ID},
		},
		{
			handler: "handler2",
			headers: map[string][]string{"Content-Type": {"application/json"}},
			body:    []byte(`{"test": "data2"}`),
			linkTo:  []entity.CollectionID{collection1ID, collection2ID},
		},
		{
			handler: "handler3",
			headers: map[string][]string{"Content-Type": {"application/json"}},
			body:    []byte(`{"test": "data3"}`),
			linkTo:  []entity.CollectionID{collection2ID},
		},
	}

	// Insert requests
	for i := range requests {
		headersJSON, err := json.Marshal(requests[i].headers)
		require.NoError(t, err)

		sql := pgh.Builder().
			Insert("requests").
			Columns("handler", "headers", "body").
			Values(requests[i].handler, headersJSON, requests[i].body).
			Suffix("RETURNING id")

		var requestID int64
		require.NoError(t, px.SelectOne(ctx, s.conn(ctx), sql, &requestID))
		requests[i].requestID = requestID

		// Link to collections
		for _, collectionID := range requests[i].linkTo {
			_, err = px.Exec(ctx, s.conn(ctx),
				pgh.Builder().Insert("request_collections").
					Columns("request_id", "collection_id").
					Values(requestID, collectionID),
			)
			require.NoError(t, err)
		}
	}

	// Verify initial state
	sql := pgh.Builder().
		Select("COUNT(*)").
		From("request_collections")

	var requestCollectionCount int
	require.NoError(t, px.SelectOne(ctx, s.conn(ctx), sql, &requestCollectionCount))
	require.Equal(t, 4, requestCollectionCount) // 1 + 2 + 1 links

	sql = pgh.Builder().
		Select("COUNT(*)").
		From("requests")

	var requestCount int
	require.NoError(t, px.SelectOne(ctx, s.conn(ctx), sql, &requestCount))
	require.Equal(t, 3, requestCount)

	// Clean collections
	err := s.CleanDatabase(ctx, []entity.CollectionID{collection1ID, collection2ID})
	require.NoError(t, err)

	// Verify collections are deleted
	sql = pgh.Builder().
		Select("*").
		From("collections").
		Where(sq.Eq{"id": []entity.CollectionID{collection1ID, collection2ID}})

	var collections []dbmodel.Collection
	require.NoError(t, px.Select(ctx, s.conn(ctx), sql, &collections))
	require.Empty(t, collections)

	// Verify request-collection links are deleted
	sql = pgh.Builder().
		Select("COUNT(*)").
		From("request_collections")

	require.NoError(t, px.SelectOne(ctx, s.conn(ctx), sql, &requestCollectionCount))
	require.Zero(t, requestCollectionCount)

	// Verify requests are preserved
	sql = pgh.Builder().
		Select("COUNT(*)").
		From("requests")

	require.NoError(t, px.SelectOne(ctx, s.conn(ctx), sql, &requestCount))
	require.Equal(t, 0, requestCount) // All requests should still be there
}
