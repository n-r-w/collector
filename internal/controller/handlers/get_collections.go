package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/pb/api/collector"
	"github.com/n-r-w/ctxlog"
	"github.com/samber/mo"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// GetCollections implements collector.CollectionServiceServer.
func (s *Service) GetCollections(
	ctx context.Context, req *collector.GetCollectionsRequest,
) (*collector.GetCollectionsResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, invalidRequestError(err)
	}

	filter, err := convertCollectionFilterToEntity(req)
	if err != nil {
		return nil, invalidRequestError(err)
	}

	collections, err := s.collectionManager.GetCollections(ctx, filter)
	if err != nil {
		ctxlog.Error(ctx, "failed to list collections", slog.Any("error", err))
		return nil, grpc_status.Errorf(codes.Internal, "failed to list collections: %v", err)
	}

	protoCollections := make([]*collector.Collection, 0, len(collections))
	for _, c := range collections {
		protoCollections = append(protoCollections, convertCollectionFromEntity(c))
	}

	ctxlog.Debug(ctx, "listed collections", slog.Int("count", len(protoCollections)))

	return &collector.GetCollectionsResponse{
		Collections: protoCollections,
	}, nil
}

func convertCollectionStatusFromEntity(status entity.CollectionStatus) collector.Status {
	switch status {
	case entity.StatusPending:
		return collector.Status_STATUS_PENDING
	case entity.StatusInProgress:
		return collector.Status_STATUS_IN_PROGRESS
	case entity.StatusFinalizing:
		return collector.Status_STATUS_FINALIZING
	case entity.StatusCompleted:
		return collector.Status_STATUS_COMPLETED
	case entity.StatusFailed:
		return collector.Status_STATUS_FAILED
	case entity.StatusCancelled:
		return collector.Status_STATUS_CANCELLED
	case entity.StatusUnknown:
		return collector.Status_STATUS_UNSPECIFIED
	}

	return collector.Status_STATUS_UNSPECIFIED
}

func convertCollectionFromEntity(collection entity.Collection) *collector.Collection {
	protoStatus := &collector.Collection{ //exhaustruct:enforce
		CollectionId: int64(collection.ID),
		Status:       convertCollectionStatusFromEntity(collection.Status),
		StartedAt:    timeToProtoPtr(collection.StartedAt.ToPointer()),
		CompletedAt:  timeToProtoPtr(collection.CompletedAt.ToPointer()),
		ErrorMessage: collection.ErrorMessage.OrEmpty(),
		RequestCount: uint64(collection.RequestCount), //nolint:gosec // ok
		Task:         convertTaskFromEntity(collection.Task),
		ResultId:     string(collection.ResultID.OrEmpty()),
	}

	return protoStatus
}

func convertTaskFromEntity(task entity.Task) *collector.Task {
	return &collector.Task{ //exhaustruct:enforce
		MessageSelection: convertMessageSelectionCriteriaFromEntity(task.MessageSelection),
		Completion:       convertCompletionCriteriaFromEntity(task.Completion),
	}
}

func convertMessageSelectionCriteriaFromEntity(
	criteria entity.MessageSelectionCriteria,
) *collector.MessageSelectionCriteria {
	return &collector.MessageSelectionCriteria{ //exhaustruct:enforce
		Handler:        criteria.Handler,
		HeaderCriteria: convertHeaderCriteriaFromEntity(criteria.HeaderCriteria),
	}
}

func convertHeaderCriteriaFromEntity(criteria []entity.HeaderCriteria) []*collector.Header {
	result := make([]*collector.Header, 0, len(criteria))
	for _, c := range criteria {
		result = append(result, &collector.Header{ //exhaustruct:enforce
			HeaderName: c.HeaderName,
			Pattern:    c.Pattern.String(),
		})
	}

	return result
}

func convertCompletionCriteriaFromEntity(criteria entity.CompletionCriteria) *collector.CompletionCriteria {
	return &collector.CompletionCriteria{ //exhaustruct:enforce
		TimeLimit:         durationpb.New(criteria.TimeLimit),
		RequestCountLimit: uint32(criteria.RequestCountLimit), //nolint:gosec // ok
	}
}

func convertCollectionFilterToEntity(req *collector.GetCollectionsRequest) (entity.CollectionFilter, error) {
	statuses, err := convertCollectionStatusesToEntity(req.GetStatuses())
	if err != nil {
		return entity.CollectionFilter{}, err
	}

	return entity.CollectionFilter{
		Statuses: statuses,
		FromTime: mo.PointerToOption(timeFromProto(req.GetFromTime())),
		ToTime:   mo.PointerToOption(timeFromProto(req.GetToTime())),
	}, nil
}

func convertCollectionStatusToEntity(status collector.Status) (entity.CollectionStatus, error) {
	switch status {
	case collector.Status_STATUS_PENDING:
		return entity.StatusPending, nil
	case collector.Status_STATUS_IN_PROGRESS:
		return entity.StatusInProgress, nil
	case collector.Status_STATUS_FINALIZING:
		return entity.StatusFinalizing, nil
	case collector.Status_STATUS_COMPLETED:
		return entity.StatusCompleted, nil
	case collector.Status_STATUS_FAILED:
		return entity.StatusFailed, nil
	case collector.Status_STATUS_CANCELLED:
		return entity.StatusCancelled, nil
	case collector.Status_STATUS_UNSPECIFIED:
		return entity.StatusUnknown, errors.New("status is unspecified")
	default:
		return entity.StatusUnknown, fmt.Errorf("unknown status: %v", status)
	}
}

func convertCollectionStatusesToEntity(statuses []collector.Status) ([]entity.CollectionStatus, error) {
	collectionStatuses := make([]entity.CollectionStatus, 0, len(statuses))
	for _, status := range statuses {
		s, err := convertCollectionStatusToEntity(status)
		if err != nil {
			return nil, err
		}
		collectionStatuses = append(collectionStatuses, s)
	}

	return collectionStatuses, nil
}
