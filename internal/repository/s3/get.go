package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3_api "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
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

			if !s.processChunk(ctx, resultID, offset, end, chunksChan) {
				return
			}
		}
	}()

	return chunksChan, nil
}

func (s *Service) processChunk(
	ctx context.Context,
	resultID entity.ResultID,
	offset, end int64,
	chunksChan chan<- entity.RequestChunk,
) bool {
	getResp, err := s.client.GetObject(ctx, &s3_api.GetObjectInput{
		Bucket: aws.String(s.cfg.S3.Bucket),
		Key:    aws.String(string(resultID)),
		Range:  aws.String(fmt.Sprintf("bytes=%d-%d", offset, end)),
	})
	if err != nil {
		chunksChan <- entity.RequestChunk{
			Err: fmt.Errorf("failed to get object: %w", err),
		}
		return false
	}

	// Create a buffer to read data in chunks
	buffer := make([]byte, s.cfg.S3.ReadChunkSize)
	reader := getResp.Body
	defer ctxlog.CloseError(ctx, reader)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			// Create a copy of the data to avoid buffer reuse issues
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			chunksChan <- entity.RequestChunk{
				Data: chunk,
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			chunksChan <- entity.RequestChunk{
				Err: fmt.Errorf("failed to read object: %w", err),
			}
			return false
		}
	}

	return true
}
