package entity

import "errors"

var (
	// ErrCollectionNotFound indicates that requested collection doesn't exist.
	ErrCollectionNotFound = errors.New("collection not found")
	// ErrInvalidStatus indicates that collection status is invalid.
	ErrInvalidStatus = errors.New("invalid collection status")
	// ErrInvalidArchiveContent indicates that the content to be archived is invalid.
	ErrInvalidArchiveContent = errors.New("invalid archive content")
)
