package consumer

import (
	"context"

	"github.com/n-r-w/collector/internal/entity"
)

// IHandlers is responsible for processing incoming requests.
type IHandlers interface {
	// HandleRequest processes one or more requests and stores them in matching collections.
	HandleRequest(ctx context.Context, requests []entity.RequestContent) error
}
