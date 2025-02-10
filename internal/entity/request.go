package entity

import (
	"time"
)

// RequestContent represents stored request content.
type RequestContent struct {
	Handler   string              // HTTP/gRPC handler name
	Headers   map[string][]string // Request headers
	Body      []byte              // Request body
	CreatedAt time.Time           // Timestamp when request was received
}

// MatchResult represents a request-match result.
type MatchResult struct {
	RequestPos    int            // Position of the request in the batch
	CollectionIDs []CollectionID // IDs of collections that match the request
}

// RequestChunk is a chunk of collection results.
type RequestChunk struct {
	Data []byte
	Err  error
}
