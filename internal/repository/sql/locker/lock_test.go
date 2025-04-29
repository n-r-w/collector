package locker

import (
	"testing"

	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/collector/internal/repository/sql"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/stretchr/testify/require"
)

func TestTryLock(t *testing.T) {
	t.Parallel()

	ctx, s := sql.SetupTest(t,
		func(cfg *config.Config, db *db.PxDB, txmgr *txmgr.TransactionManager) (*Service, error) {
			cfg.App.EnvType = ctxlog.EnvDevelopment
			return New(cfg, db, txmgr)
		},
	)

	// *** successful lock acquisition
	lockKey1 := entity.LockKey(1)

	// Acquire lock
	ctxLock1, unlocker1, err := s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock1)
	require.NotNil(t, unlocker1)

	// Release lock
	require.NoError(t, unlocker1.Unlock(ctx))

	// *** unsuccessful lock acquisition
	lockKey1 = entity.LockKey(2)

	// Acquire lock
	ctxLock1, unlocker1, err = s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock1)
	require.NotNil(t, unlocker1)

	// Acquire lock again
	ctxLock2, unlocker2, err := s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock2)
	require.Nil(t, unlocker2)

	// Release first lock
	require.NoError(t, unlocker1.Unlock(ctx))

	// Acquire lock again
	ctxLock2, unlocker2, err = s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock2)
	require.NotNil(t, unlocker2)

	// Release second lock
	require.NoError(t, unlocker2.Unlock(ctx))

	// ***  reacquire after unlock
	lockKey1 = entity.LockKey(3)
	// First acquisition
	ctxLock1, unlocker1, err = s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock1)

	// Release first lock
	require.NoError(t, unlocker1.Unlock(ctx))

	// Second acquisition should succeed
	ctxLock2, unlocker2, err = s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock2)

	// Release second lock
	require.NoError(t, unlocker2.Unlock(ctx))

	// *** different keys can be locked simultaneously
	lockKey1 = entity.LockKey(4)
	lockKey2 := entity.LockKey(5)

	// Acquire first lock
	ctxLock1, unlocker1, err = s.TryLock(ctx, lockKey1)
	require.NoError(t, err)
	require.NotNil(t, ctxLock1)

	// Acquire second lock
	ctxLock2, unlocker2, err = s.TryLock(ctx, lockKey2)
	require.NoError(t, err)
	require.NotNil(t, ctxLock2)

	// Release both locks
	require.NoError(t, unlocker1.Unlock(ctx))
	require.NoError(t, unlocker2.Unlock(ctx))
}
