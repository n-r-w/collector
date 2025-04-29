package locker

import (
	"context"
	"errors"
	"fmt"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// TryLockFunc executes a function with a lock.
func (s *Service) TryLockFunc(
	ctx context.Context, key entity.LockKey, fn func(context.Context) error,
) (acquired bool, err error) {
	var (
		ctxLock  context.Context
		unlocker entity.IUnlocker
	)
	ctxLock, unlocker, err = s.TryLock(ctx, key)
	if err != nil {
		return false, err
	}

	if unlocker == nil {
		return false, nil
	}

	defer func() {
		unlockErr := unlocker.Unlock(ctxLock)
		if unlockErr != nil {
			err = errors.Join(err, unlockErr)
		}
	}()

	return true, fn(ctxLock)
}

// TryLock tries to get a lock using pg_try_advisory_lock.
func (s *Service) TryLock(
	ctx context.Context, key entity.LockKey,
) (ctxLock context.Context, u entity.IUnlocker, err error) {
	var finisher txmgr.ITransactionFinisher
	ctxLock, finisher, err = s.txManager.BeginTx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("TryLock: failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = finisher.Rollback(ctx)
		}
	}()

	var acquired bool
	if err := px.SelectOnePlain(
		ctx, s.conn(ctxLock), `SELECT pg_try_advisory_xact_lock($1)`, &acquired, pgh.Args{key}); err != nil {
		return nil, nil, fmt.Errorf("TryLock: failed to acquire lock: %w", err)
	}

	if !acquired {
		_ = finisher.Rollback(ctx)
		return ctxLock, nil, nil
	}

	return ctxLock, &unlocker{tx: finisher}, nil
}

// unlocker implements entity.IUnlocker.
type unlocker struct {
	tx txmgr.ITransactionFinisher
}

// Unlock implements entity.IUnlocker Unlock method.
func (u *unlocker) Unlock(ctx context.Context) error {
	if u.tx == nil {
		return nil
	}

	_ = u.tx.Commit(ctx)
	return nil
}
