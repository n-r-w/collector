package entity

import (
	"regexp"
	"time"
)

// Task contains parameters for creating a new collection.
type Task struct {
	MessageSelection MessageSelectionCriteria
	Completion       CompletionCriteria
}

// MessageSelectionCriteria defines criteria for selecting messages to collect.
type MessageSelectionCriteria struct {
	// Handler is the HTTP/gRPC handler to match.
	Handler string
	// HeaderCriteria is a list of header criteria to match against request headers.
	HeaderCriteria []HeaderCriteria
}

// HeaderCriteria defines a single header matching Criteria.
type HeaderCriteria struct {
	// HeaderName is the name of the HTTP header to match.
	HeaderName string
	// Pattern is the regular expression to match against the header value.
	Pattern *regexp.Regexp
}

// CompletionCriteria defines when to complete the collection.
type CompletionCriteria struct {
	// TimeLimit defines the maximum duration for collection.
	TimeLimit time.Duration
	// RequestCountLimit defines the maximum number of requests to collect.
	RequestCountLimit int
}
