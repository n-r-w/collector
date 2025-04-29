package s3

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3_api "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/ctxlog"
)

// CleanObjectStorage deletes the result of a collection.
func (s *Service) CleanObjectStorage(ctx context.Context, resultIDs []entity.ResultID) error {
	// TODO: DeleteObjects causes a `MissingContentMD5` error and how to fix it is not clear yet
	for _, id := range resultIDs {
		_, err := s.client.DeleteObject(ctx, &s3_api.DeleteObjectInput{
			Bucket: aws.String(s.cfg.S3.Bucket),
			Key:    aws.String(string(id)),
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			ctxlog.Error(ctx, "failed to delete object from S3", slog.Any("id", id), slog.Any("error", err))
		}
	}
	return nil
}
