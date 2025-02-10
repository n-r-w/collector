package reqprocessor

import (
	"context"
	"fmt"
	"strings"

	"github.com/n-r-w/ammo-collector/internal/controller/consumer"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/bootstrap"
)

// Service implements kafka.Handlers, bootstrap.IService interfaces.
type Service struct {
	requestStorer IRequestStorer
	cacheGetter   ICollectionCacher
}

var (
	_ consumer.IHandlers = (*Service)(nil)
	_ bootstrap.IService = (*Service)(nil)
)

// New creates a new RequestProcessor instance.
func New(
	requestStorer IRequestStorer,
	cacheGetter ICollectionCacher,
) *Service {
	return &Service{
		requestStorer: requestStorer,
		cacheGetter:   cacheGetter,
	}
}

// HandleRequest processes a single request and stores it in matching collections.
func (s *Service) HandleRequest(ctx context.Context, requests []entity.RequestContent) error {
	collections := s.cacheGetter.Get()

	if len(collections) == 0 {
		return nil
	}

	// Find matching collections and store request
	toStore := make([]entity.MatchResult, 0, len(requests))
	for i, request := range requests {
		match := entity.MatchResult{RequestPos: i}
		for _, collection := range collections {
			if collection.Status != entity.StatusPending && collection.Status != entity.StatusInProgress {
				continue
			}

			if s.matchesCriteria(request, collection.Task.MessageSelection) {
				match.CollectionIDs = append(match.CollectionIDs, collection.ID)
			}
		}

		if len(match.CollectionIDs) > 0 {
			toStore = append(toStore, match)
		}
	}

	if len(toStore) == 0 {
		return nil
	}

	// Store request
	if err := s.storeRequest(ctx, requests, toStore); err != nil {
		return fmt.Errorf("failed to store request: %w", err)
	}

	return nil
}

// matchesCriteria checks if the request matches the collection criteria.
func (s *Service) matchesCriteria(request entity.RequestContent, criteria entity.MessageSelectionCriteria,
) bool {
	if !strings.EqualFold(request.Handler, criteria.Handler) {
		return false
	}

	// if no headers are specified, the request is considered matching
	if len(criteria.HeaderCriteria) == 0 {
		return true
	}

	// if at least one header matches, the request is considered matching
	for header, values := range request.Headers {
		for _, ch := range criteria.HeaderCriteria {
			if !strings.EqualFold(header, ch.HeaderName) {
				continue
			}

			for _, value := range values {
				if ch.Pattern.MatchString(value) {
					return true
				}
			}
		}
	}

	return false
}

// storeRequest stores the request in the collection and updates collection counters.
func (s *Service) storeRequest(
	ctx context.Context, requests []entity.RequestContent, toStore []entity.MatchResult,
) error {
	// Store request
	if err := s.requestStorer.Store(ctx, requests, toStore); err != nil {
		return fmt.Errorf("failed to store request: %w", err)
	}

	return nil
}

// Info returns service info. Implements bootstrap.IService Info method.
func (s *Service) Info() bootstrap.Info {
	return bootstrap.Info{
		Name: "Request Processor",
	}
}

// Start starts the service. Implements bootstrap.IService Start method.
func (s *Service) Start(_ context.Context) error {
	return nil
}

// Stop stops the service. Implements bootstrap.IService Stop method.
func (s *Service) Stop(_ context.Context) error {
	return nil
}
