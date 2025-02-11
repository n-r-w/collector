package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/pb/api/collector"
	"github.com/n-r-w/ctxlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateTask implements collector.CollectorServer.
func (s *Service) CreateTask(
	ctx context.Context, req *collector.CreateTaskRequest,
) (*collector.CreateTaskResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, invalidRequestError(err)
	}

	if int(req.GetCompletionCriteria().GetRequestCountLimit()) > s.maxRequestsPerCollection {
		return nil, invalidRequestError(
			fmt.Errorf("request count limit must be less than %d", s.maxRequestsPerCollection))
	}

	headerCriteria, err := s.convertHeaderCriteria(req.GetSelectionCriteria().GetHeaderCriteria())
	if err != nil {
		return nil, invalidRequestError(err)
	}

	task := entity.Task{
		MessageSelection: entity.MessageSelectionCriteria{
			Handler:        req.GetSelectionCriteria().GetHandler(),
			HeaderCriteria: headerCriteria,
		},
		Completion: entity.CompletionCriteria{
			TimeLimit:         req.GetCompletionCriteria().GetTimeLimit().AsDuration(),
			RequestCountLimit: int(req.GetCompletionCriteria().GetRequestCountLimit()),
		},
	}

	collectionID, err := s.collectionManager.CreateCollection(ctx, task)
	if err != nil {
		ctxlog.Error(ctx, "failed to create collection", slog.Any("error", err))
		return nil, status.Errorf(codes.Internal, "failed to create collection: %v", err)
	}

	ctxlog.Debug(ctx, "collection created", slog.String("collection_id", collectionID.String()))

	return &collector.CreateTaskResponse{
		CollectionId: int64(collectionID),
	}, nil
}

func (s *Service) convertHeaderCriteria(criteria []*collector.Header) ([]entity.HeaderCriteria, error) {
	result := make([]entity.HeaderCriteria, 0, len(criteria))
	for _, c := range criteria {
		pattern, err := regexp.Compile(c.GetPattern())
		if err != nil {
			return nil, fmt.Errorf("compile regexp: %w", err)
		}

		result = append(result, entity.HeaderCriteria{
			HeaderName: c.GetHeaderName(),
			Pattern:    pattern,
		})
	}
	return result, nil
}
