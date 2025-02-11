package handlers

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/pb/api/collector"
	"github.com/n-r-w/grpcsrv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service is a gRPC handlers implementation.
type Service struct {
	collector.UnimplementedCollectionServiceServer

	collectionManager        ICollectionManager
	resultGetter             IResultGetter
	maxRequestsPerCollection int
}

var (
	_ collector.CollectionServiceServer = (*Service)(nil)
	_ grpcsrv.IGRPCInitializer          = (*Service)(nil)
)

// New creates a new gRPC handlers implementation.
func New(
	cfg *config.Config,
	collectionManager ICollectionManager,
	resultGetter IResultGetter,
) *Service {
	return &Service{
		collectionManager:        collectionManager,
		resultGetter:             resultGetter,
		maxRequestsPerCollection: cfg.Collection.MaxRequestsPerCollection,
	}
}

func timeToProtoPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timeFromProto(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}

	tt := t.AsTime()
	if tt.IsZero() {
		return nil
	}

	return &tt
}

func (s *Service) RegisterGRPCServer(srv *grpc.Server) {
	collector.RegisterCollectionServiceServer(srv, s)
}

func (s *Service) RegisterHTTPHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return collector.RegisterCollectionServiceHandler(ctx, mux, conn)
}

func (s *Service) GetOptions() grpcsrv.InitializeOptions {
	return grpcsrv.InitializeOptions{
		HTTPHandlerRequired: true,
	}
}

func invalidRequestError(err error) error {
	return grpc_status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
}
