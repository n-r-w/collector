package colmanager

import (
	"regexp"
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
	sq "github.com/n-r-w/squirrel"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
)

/*func TestGetActiveCollections(t *testing.T) {
	t.Parallel()

	ctx, s := setupTest(t)

	// Create test task
	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler: "test-handler",
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         time.Hour,
			RequestCountLimit: 100,
		},
	}

	// Create collections with different statuses
	pendingID, err := s.CreateCollection(ctx, task)
	require.NoError(t, err)

	runningID, err := s.CreateCollection(ctx, task)
	require.NoError(t, err)

	completedID, err := s.CreateCollection(ctx, task)
	require.NoError(t, err)

	// Update statuses and add some request counts
	_, err = px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusPending).
			Where("id = ?", pendingID))
	require.NoError(t, err)

	_, err = px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusInProgress).
			Set("request_count", 10).
			Set("started_at", time.Now()).
			Set("updated_at", time.Now()).
			Where("id = ?", runningID))
	require.NoError(t, err)

	_, err = px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusCompleted).
			Where("id = ?", completedID))
	require.NoError(t, err)

	// Get active collections
	collections, err := s.GetActiveCollections(ctx)
	require.NoError(t, err)
	require.Len(t, collections, 2) // Should only return pending and running

	// Map collections by ID for easier verification
	collectionsByID := make(map[entity.CollectionID]entity.Collection)
	for _, c := range collections {
		collectionsByID[c.ID] = c
	}

	// Verify pending collection
	pending, ok := collectionsByID[pendingID]
	require.True(t, ok)
	require.Equal(t, entity.StatusPending, pending.Status)
	require.EqualValues(t, 0, pending.RequestCount)
	require.False(t, pending.StartedAt.IsSome())
	require.False(t, pending.CompletedAt.IsSome())

	// Verify running collection
	running, ok := collectionsByID[runningID]
	require.True(t, ok)
	require.Equal(t, entity.StatusInProgress, running.Status)
	require.EqualValues(t, 10, running.RequestCount)
	require.True(t, running.StartedAt.IsSome())
	require.False(t, running.CompletedAt.IsSome())

	// Verify completed collection
	_, ok = collectionsByID[completedID]
	require.False(t, ok)
}*/

func TestGetCollection(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, _ *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db)
		},
	)

	// Create test task
	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler: "test-handler",
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         time.Hour,
			RequestCountLimit: 100,
		},
	}

	// Create a test collection
	collectionID, err := s.CreateCollection(ctx, task)
	require.NoError(t, err)

	// Update collection with some data
	startedAt := time.Now()
	updatedAt := startedAt.Add(time.Minute)
	_, err = px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusInProgress).
			Set("request_count", 42).
			Set("started_at", startedAt).
			Set("updated_at", updatedAt).
			Where(sq.Eq{"id": collectionID}))
	require.NoError(t, err)

	// Test getting the collection
	collection, err := s.GetCollection(ctx, collectionID)
	require.NoError(t, err)

	// Verify collection fields
	require.Equal(t, collectionID, collection.ID)
	require.Equal(t, entity.StatusInProgress, collection.Status)
	require.EqualValues(t, 42, collection.RequestCount)
	require.True(t, collection.StartedAt.IsPresent())
	require.True(t, collection.UpdatedAt.IsPresent())
	require.False(t, collection.CompletedAt.IsPresent())
	require.WithinDuration(t, startedAt.UTC(), collection.StartedAt.OrEmpty().UTC(), time.Millisecond)
	require.WithinDuration(t, updatedAt.UTC(), collection.UpdatedAt.OrEmpty().UTC(), time.Millisecond)

	// Test getting non-existent collection
	nonExistentID := entity.CollectionID(999999)
	_, err = s.GetCollection(ctx, nonExistentID)
	require.ErrorIs(t, err, entity.ErrCollectionNotFound)
	require.ErrorContains(t, err, "GetCollection: id 999999")
}

func TestGetCollections(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, _ *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db)
		},
	)

	// Create test task
	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler: "test-handler",
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         time.Hour,
			RequestCountLimit: 100,
		},
	}

	// Create collections with different statuses and times
	baseTime := time.Now().UTC()
	timeDay1 := baseTime.Add(-24 * time.Hour)
	timeDay2 := baseTime.Add(-48 * time.Hour)
	timeDay3 := baseTime.Add(-72 * time.Hour)

	// Create collections for different days
	collectionIDs := make([]entity.CollectionID, 0, 3)
	for _, createTime := range []time.Time{timeDay1, timeDay2, timeDay3} {
		id, err := s.CreateCollection(ctx, task)
		require.NoError(t, err)
		collectionIDs = append(collectionIDs, id)

		// Set created_at time
		_, err = px.Exec(ctx, s.conn(ctx),
			pgh.Builder().Update("collections").
				Set("created_at", createTime).
				Where(sq.Eq{"id": id}))
		require.NoError(t, err)
	}

	// Set different statuses
	_, err := px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusPending).
			Where(sq.Eq{"id": collectionIDs[0]}))
	require.NoError(t, err)

	_, err = px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusInProgress).
			Set("request_count", 42).
			Set("started_at", timeDay1).
			Set("updated_at", timeDay1.Add(time.Minute)).
			Where(sq.Eq{"id": collectionIDs[1]}))
	require.NoError(t, err)

	_, err = px.Exec(ctx, s.conn(ctx),
		pgh.Builder().Update("collections").
			Set("status", entity.StatusCompleted).
			Set("completed_at", timeDay3).
			Where(sq.Eq{"id": collectionIDs[2]}))
	require.NoError(t, err)

	// Test 1: Get all collections without filter
	collections, err := s.GetCollections(ctx, entity.CollectionFilter{})
	require.NoError(t, err)
	require.Len(t, collections, 3)

	// Test 2: Filter by status
	collections, err = s.GetCollections(ctx, entity.CollectionFilter{
		Statuses: []entity.CollectionStatus{entity.StatusPending, entity.StatusInProgress},
	})
	require.NoError(t, err)
	require.Len(t, collections, 2)
	for _, c := range collections {
		require.Contains(t, []entity.CollectionStatus{entity.StatusPending, entity.StatusInProgress}, c.Status)
	}

	// Test 3: Filter by time range
	collections, err = s.GetCollections(ctx, entity.CollectionFilter{
		FromTime: mo.Some(timeDay2.Add(-time.Hour)),
		ToTime:   mo.Some(timeDay2.Add(time.Hour)),
	})
	require.NoError(t, err)
	require.Len(t, collections, 1)
	require.Equal(t, collectionIDs[1], collections[0].ID)

	// Test 4: Filter by both status and time range
	collections, err = s.GetCollections(ctx, entity.CollectionFilter{
		Statuses: []entity.CollectionStatus{entity.StatusCompleted},
		FromTime: mo.Some(timeDay3.Add(-time.Hour)),
		ToTime:   mo.Some(timeDay3.Add(time.Hour)),
	})
	require.NoError(t, err)
	require.Len(t, collections, 1)
	require.Equal(t, entity.StatusCompleted, collections[0].Status)
	require.Equal(t, collectionIDs[2], collections[0].ID)

	// Test 5: Empty result with non-matching filter
	collections, err = s.GetCollections(ctx, entity.CollectionFilter{
		Statuses: []entity.CollectionStatus{entity.StatusFailed},
	})
	require.NoError(t, err)
	require.Empty(t, collections)
}

func TestCreateCollection(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, _ *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db)
		},
	)

	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler: "handler-name",
			HeaderCriteria: []entity.HeaderCriteria{
				{
					HeaderName: "header-name",
					Pattern:    regexp.MustCompile(`^application/json.*$`),
				},
			},
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         time.Hour,
			RequestCountLimit: 100,
		},
	}

	id, err := s.CreateCollection(ctx, task)
	require.NoError(t, err)
	require.NotZero(t, id)

	dbCollection, err := dbmodel.CollectionByID(ctx, s.conn(ctx), int64(id))
	require.NoError(t, err)

	require.EqualValues(t, id, dbCollection.ID)
	require.EqualValues(t, entity.StatusPending, dbCollection.Status)
	require.EqualValues(t, task.Completion.RequestCountLimit, dbCollection.RequestCountLimit)
	require.EqualValues(t, task.Completion.TimeLimit, dbCollection.RequestDurationLimit)
	require.EqualValues(t, 0, dbCollection.RequestCount)
	require.False(t, dbCollection.CreatedAt.IsZero())
	require.False(t, dbCollection.StartedAt.Valid)
	require.False(t, dbCollection.UpdatedAt.Valid)
	require.False(t, dbCollection.CompletedAt.Valid)
	require.False(t, dbCollection.ErrorMessage.Valid)
	require.False(t, dbCollection.ErrorCode.Valid)
	require.False(t, dbCollection.ResultID.Valid)

	expectedCriteria := `
{  
"handler": "handler-name",
"headerCriteria": [
	{
	"pattern": "^application/json.*$",
	"headerName": "header-name"
	}
]  
}`

	require.JSONEq(t, expectedCriteria, string(dbCollection.Criteria))
}
