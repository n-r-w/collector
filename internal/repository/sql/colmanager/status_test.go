package colmanager

import (
	"testing"
	"time"

	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/collector/internal/repository/sql"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/txmgr"
	sq "github.com/n-r-w/squirrel"
	"github.com/stretchr/testify/require"
)

func TestUpdateStatus(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, _ *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db)
		},
	)

	// Create a test task
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

	// Test 1: Update to a valid non-terminal status
	beforeUpdate := time.Now().UTC()
	err = s.UpdateStatus(ctx, collectionID, entity.StatusInProgress)
	require.NoError(t, err)

	// Verify the update
	var data struct {
		Status      entity.CollectionStatus
		UpdatedAt   time.Time
		CompletedAt *time.Time
	}
	err = px.SelectOne(ctx, s.conn(ctx),
		pgh.Builder().Select("status", "updated_at", "completed_at").
			From("collections").
			Where(sq.Eq{"id": collectionID}),
		&data)
	require.NoError(t, err)
	require.Equal(t, entity.StatusInProgress, data.Status)
	require.True(t, data.UpdatedAt.After(beforeUpdate))
	require.Nil(t, data.CompletedAt)

	// Test 2: Update to a terminal status
	beforeUpdate = time.Now().UTC()
	err = s.UpdateStatus(ctx, collectionID, entity.StatusCompleted)
	require.NoError(t, err)

	// Verify the terminal status update
	err = px.SelectOne(ctx, s.conn(ctx),
		pgh.Builder().Select("status", "updated_at", "completed_at").
			From("collections").
			Where(sq.Eq{"id": collectionID}),
		&data)
	require.NoError(t, err)
	require.Equal(t, entity.StatusCompleted, data.Status)
	require.True(t, data.UpdatedAt.After(beforeUpdate))
	require.NotNil(t, data.CompletedAt)
	require.True(t, data.CompletedAt.After(beforeUpdate))

	// Test 3: Try to update non-existent collection
	err = s.UpdateStatus(ctx, entity.CollectionID(999999), entity.StatusInProgress)
	require.ErrorIs(t, err, entity.ErrCollectionNotFound)

	// Test 4: Try to update with invalid status
	err = s.UpdateStatus(ctx, collectionID, entity.CollectionStatus(999))
	require.ErrorIs(t, err, entity.ErrInvalidStatus)
}
