package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3_api "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n-r-w/ammo-collector/internal/entity"
)

// GetResult returns the result of a collection. Implements apiprocessor.IResultGetter.GetResult.
func (s *Service) GetResult(ctx context.Context, resultID entity.ResultID) (<-chan entity.RequestChunk, error) {
	// Get object file size
	headResp, err := s.client.HeadObject(ctx, &s3_api.HeadObjectInput{
		Bucket: aws.String(s.cfg.S3.Bucket),
		Key:    aws.String(string(resultID)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}
	fileSize := headResp.ContentLength

	// Return data stream
	chunksChan := make(chan entity.RequestChunk)
	if fileSize == nil || *fileSize == 0 {
		close(chunksChan)
		return chunksChan, nil
	}

	chunkSize := int64(s.cfg.S3.ReadChunkSize)
	go func() {
		defer close(chunksChan)

		for offset := int64(0); offset < *fileSize; offset += chunkSize {
			end := offset + chunkSize - 1
			if end >= *fileSize {
				end = *fileSize - 1
			}

			getResp, err := s.client.GetObject(ctx, &s3_api.GetObjectInput{
				Bucket: aws.String(s.cfg.S3.Bucket),
				Key:    aws.String(string(resultID)),
				Range:  aws.String(fmt.Sprintf("bytes=%d-%d", offset, end)),
			})
			if err != nil {
				chunksChan <- entity.RequestChunk{
					Err: fmt.Errorf("failed to get object: %w", err),
				}
				return
			}

			data, err := io.ReadAll(getResp.Body)
			if err != nil {
				chunksChan <- entity.RequestChunk{
					Err: fmt.Errorf("failed to read object: %w", err),
				}
				return
			}

			chunksChan <- entity.RequestChunk{
				Data: data,
			}
		}
	}()

	return chunksChan, nil
}
