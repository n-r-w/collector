package handlers

import (
	"errors"
	"fmt"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/collector/internal/pb/api/collector"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"
)

// GetResult returns the result of a collection as a stream of bytes.
func (s *Service) GetResult(
	req *collector.GetResultRequest, stream grpc.ServerStreamingServer[collector.GetResultResponse],
) error {
	if err := req.ValidateAll(); err != nil {
		return invalidRequestError(err)
	}

	ctx := stream.Context()

	// Get the result chunks channel from the result getter
	resultChan, err := s.resultGetter.GetResult(ctx, entity.CollectionID(req.GetCollectionId()))
	if err != nil {
		if errors.Is(err, entity.ErrCollectionNotFound) {
			return grpc_status.Error(codes.NotFound, err.Error())
		} else if errors.Is(err, entity.ErrInvalidStatus) {
			return grpc_status.Error(codes.FailedPrecondition, err.Error())
		}

		return grpc_status.Error(codes.Internal, fmt.Sprintf("failed to get result: %v", err))
	}

	// Stream each chunk to the client
	for chunk := range resultChan {
		// Check if there was an error getting the chunk
		if chunk.Err != nil {
			return grpc_status.Error(codes.Internal, fmt.Sprintf("failed to get result chunk: %v", chunk.Err))
		}

		// Create and send the response
		resp := &collector.GetResultResponse{
			Content: chunk.Data,
		}

		if err := stream.Send(resp); err != nil {
			return grpc_status.Error(codes.Internal, fmt.Sprintf("failed to send result chunk: %v", err))
		}
	}

	return nil
}
