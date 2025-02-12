package entity

import (
	"strconv"
	"time"

	"github.com/samber/mo"
)

// CollectionID is a unique identifier for a task.
type CollectionID int64

// String returns a string representation of the CollectionID.
func (id CollectionID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// ResultID is a unique identifier for a result in the storage.
type ResultID string

// Collection represents an ammo collection entity.
type Collection struct {
	// ID is a unique identifier of the collection
	ID CollectionID
	// Task is the collection creation parameters
	Task Task
	// Status represents current state of the collection
	Status CollectionStatus
	// RequestCount is the total number of requests in the collection
	RequestCount int
	// CreatedAt is the timestamp when collection was created
	CreatedAt time.Time
	// StartedAt is the timestamp when collection was started
	StartedAt mo.Option[time.Time]
	// UpdatedAt is the timestamp of the last update
	UpdatedAt mo.Option[time.Time]
	// CompletedAt is the timestamp when collection reached terminal state
	CompletedAt mo.Option[time.Time]

	// ResultID is the ID of the result in the storage
	ResultID mo.Option[ResultID]
	// ErrorMessage contains error message if collection failed
	ErrorMessage mo.Option[string]
	// ErrorCode contains error code if collection failed
	ErrorCode mo.Option[int]
}

// IsOutOfTimeLimit returns true if collection is out of time limit.
func (c *Collection) IsOutOfTimeLimit() bool {
	return time.Since(c.CreatedAt) >= c.Task.Completion.TimeLimit
}

// IsOutOfRequestLimit returns true if collection is out of request limit.
func (c *Collection) IsOutOfRequestLimit() bool {
	return c.RequestCount >= c.Task.Completion.RequestCountLimit
}

// IsTerminal returns true if collection is in terminal state.
func (c *Collection) IsTerminal() bool {
	return c.Status.IsTerminal()
}

// CanStart returns true if collection can be started.
func (c *Collection) CanStart() bool {
	return c.Status == StatusPending
}

// CanStop returns true if collection can be stopped.
func (c *Collection) CanStop() bool {
	return c.Status == StatusInProgress
}

// SetStatus updates collection status and related timestamps.
func (c *Collection) SetStatus(status CollectionStatus) error {
	if !status.IsValid() {
		return ErrInvalidStatus
	}

	now := time.Now()

	c.Status = status
	c.UpdatedAt = mo.Some(now)

	if status.IsTerminal() {
		c.CompletedAt = mo.Some(now)
	}

	return nil
}

// SetError sets error message and updates status to failed.
func (c *Collection) SetError(err string) error {
	c.ErrorMessage = mo.Some(err)
	return c.SetStatus(StatusFailed)
}

// CollectionFilter contains parameters for filtering collections.
type CollectionFilter struct {
	Statuses []CollectionStatus
	FromTime mo.Option[time.Time] // created_at
	ToTime   mo.Option[time.Time] // created_at
}
