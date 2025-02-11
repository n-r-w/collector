package handlers

import (
	"context"
	"log/slog"

	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ammo-collector/internal/pb/api/collector"
	"github.com/n-r-w/ctxlog"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
)

// GetCollection implements collector.CollectionServiceServer.
func (s *Service) GetCollection(
	ctx context.Context, req *collector.GetCollectionRequest,
) (*collector.GetCollectionResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, invalidRequestError(err)
	}

	id := entity.CollectionID(req.GetCollectionId())

	collection, err := s.collectionManager.GetCollection(ctx, id)
	if err != nil {
		ctxlog.Error(ctx, "failed to get collection", slog.Any("error", err), slog.String("collection_id", id.String()))
		return nil, grpc_status.Errorf(codes.Internal, "failed to get collection: %v", err)
	}

	ctxlog.Debug(ctx, "got collection", slog.String("collection_id", id.String()))

	return &collector.GetCollectionResponse{
		Collection: convertCollectionFromEntity(collection),
	}, nil
}
