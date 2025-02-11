package handlers

import (
	"context"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/pb/api/collector"
	"github.com/n-r-w/ctxlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CancelCollection implements collector.CollectionServiceServer.
func (s *Service) CancelCollection(
	ctx context.Context, req *collector.CancelCollectionRequest,
) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, invalidRequestError(err)
	}

	if err := s.collectionManager.CancelCollection(ctx, entity.CollectionID(req.GetCollectionId())); err != nil {
		ctxlog.Error(ctx, "failed to cancel collection", slog.Any("error", err))
		return nil, status.Errorf(codes.Internal, "failed to cancel collection: %v", err)
	}

	ctxlog.Debug(ctx, "collection cancelled", slog.Int64("collection_id", req.GetCollectionId()))

	return &emptypb.Empty{}, nil
}
