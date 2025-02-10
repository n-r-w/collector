package entity

import "context"

// *********************************************************************************
// LockKey greater than zero (collection.ID) is used to lock collection finalization
// *********************************************************************************

// LockKey is a key for locking resources.
type LockKey int64

const (
	// CleanUpLockKey is a key for cleanup lock.
	CleanUpLockKey LockKey = -1
)

// IUnlocker unlocks the database.
type IUnlocker interface {
	Unlock(ctx context.Context) error
}
