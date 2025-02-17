//nolint:gocritic // tests
package sql

import (
	"context"
	"log/slog"
	"testing"

	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/repository/sql/dbmodel"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/n-r-w/testdock/v2"
	"github.com/stretchr/testify/require"
)

// GetConn returns a function that returns a connection to the database.
func GetConn(
	cfg *config.Config, connectionGetter db.IConnectionGetter,
) func(ctx context.Context) conn.IConnection {
	return func(ctx context.Context) conn.IConnection {
		var connOpts []conn.ConnectionOption
		if cfg.App.EnvType == ctxlog.EnvDevelopment {
			connOpts = append(connOpts, conn.WithLogQueries())
		}

		return connectionGetter.Connection(ctx, connOpts...)
	}
}

// SetupTest sets up a test environment for sql repository.
func SetupTest[T any](
	t *testing.T,
	f func(*config.Config, *db.PxDB, *txmgr.TransactionManager) (*T, error),
) (context.Context, *T) {
	t.Helper()

	const postgresVersion = "17.2"

	// logger
	logger := ctxlog.Must(
		ctxlog.WithTesting(t),
		ctxlog.WithLevel(slog.LevelDebug),
	)
	// put logger into context
	ctx := ctxlog.ToContext(context.Background(), logger)

	// test database connection pool
	pool, _ := testdock.GetPgxPool(t,
		testdock.DefaultPostgresDSN,
		testdock.WithMigrations("../../../../migrations", testdock.GooseMigrateFactoryPGX),
		testdock.WithDockerImage(postgresVersion),
		testdock.WithLogger(logger),
	)

	// test database implementation
	dbImpl := db.New(db.WithPool(pool), db.WithLogQueries(), db.WithLogger(logger))
	// test transaction manager implementation
	txmgrImpl := txmgr.New(dbImpl, dbImpl)

	cfg := &config.Config{}

	s, err := f(cfg, dbImpl, txmgrImpl)
	require.NoError(t, err)

	return ctx, s
}

// CreateTestCollection creates a test collection with the given task.
func CreateTestCollection(
	t *testing.T,
	ctx context.Context,
	c func(context.Context) conn.IConnection,
	task entity.Task,
) entity.CollectionID {
	t.Helper()

	criteria, err := ConvertTaskToCriteriaDB(task)
	require.NoError(t, err)

	dbCollection := dbmodel.Collection{
		Status:               int(entity.StatusPending),
		RequestCountLimit:    task.Completion.RequestCountLimit,
		RequestDurationLimit: task.Completion.TimeLimit,
		Criteria:             criteria,
	}

	require.NoError(t, dbCollection.Insert(ctx, c(ctx)))

	return entity.CollectionID(dbCollection.ID)
}
