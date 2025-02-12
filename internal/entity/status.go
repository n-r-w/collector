package entity

// CollectionStatus represents possible collection states.
type CollectionStatus int

const (
	// StatusUnknown represents an invalid or unknown status.
	StatusUnknown CollectionStatus = iota
	// StatusPending indicates that collection is created but not yet started.
	StatusPending
	// StatusInProgress indicates that collection is currently running.
	StatusInProgress
	// StatusFinalizing indicates that collection is in process of finalizing.
	StatusFinalizing
	// StatusCompleted indicates that collection has finished successfully.
	StatusCompleted
	// StatusFailed indicates that collection has failed.
	StatusFailed
	// StatusCancelled indicates that collection was cancelled by user.
	StatusCancelled
)

var statusNames = [...]string{ //nolint:gochecknoglobals // ok
	"unknown",
	"pending",
	"progress",
	"finalizing",
	"completed",
	"failed",
	"cancelled",
}

func (s CollectionStatus) String() string {
	if !s.IsValid() {
		return statusNames[StatusUnknown]
	}

	return statusNames[s]
}

// IsValid checks if the status is one of the defined constants.
func (s CollectionStatus) IsValid() bool {
	return s > StatusUnknown && s <= StatusCancelled
}

// IsTerminal returns true if the status represents a terminal state.
func (s CollectionStatus) IsTerminal() bool {
	switch s { //nolint:exhaustive // false positive
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsCollecting returns true if the status is in collecting states.
func (s CollectionStatus) IsCollecting() bool {
	return s == StatusInProgress || s == StatusPending
}

// IsFinalizing returns true if the status is in finalizing state.
func (s CollectionStatus) IsFinalizing() bool {
	return s == StatusFinalizing
}

// CollectingCollectionStatuses returns a slice of collections in collecting states.
func CollectingCollectionStatuses() []CollectionStatus {
	return []CollectionStatus{StatusInProgress, StatusPending}
}

// ActiveCollectionStatuses returns a slice of collections in active states.
func ActiveCollectionStatuses() []CollectionStatus {
	return []CollectionStatus{StatusPending, StatusInProgress, StatusFinalizing}
}

// TerminalCollectionStatuses returns a slice of collections in terminal states.
func TerminalCollectionStatuses() []CollectionStatus {
	return []CollectionStatus{StatusCompleted, StatusFailed, StatusCancelled}
}

// CollectionStatusFromInt returns the CollectionStatus corresponding to the provided integer value.
func CollectionStatusFromInt(i int) (CollectionStatus, bool) {
	//nolint:mnd // ok
	switch i {
	case 0:
		return StatusUnknown, false
	case 1:
		return StatusPending, true
	case 2:
		return StatusInProgress, true
	case 3:
		return StatusCompleted, true
	case 4:
		return StatusFailed, true
	case 5:
		return StatusCancelled, true
	default:
		return StatusUnknown, false
	}
}
