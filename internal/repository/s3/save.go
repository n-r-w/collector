package s3

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3_api "github.com/aws/aws-sdk-go-v2/service/s3"
	s3_types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/n-r-w/ctxlog"
)

// SaveResultChan is responsible for saving collection results. Implements IResultSaver.SaveResultChan.
func (s *Service) SaveResultChan(
	ctx context.Context, collectionID entity.CollectionID, requests <-chan entity.RequestChunk,
) (resultID entity.ResultID, err error) {
	// using multipart upload instead of single stream (via io.Pipe) to avoid S3 TLS requirements:
	// `unseekable stream is not supported without TLS and trailing checksum`

	defer func() {
		// drain the channel in case of errors below
		for range requests {
		}
	}()

	fileName := fmt.Sprintf("collection-%d.zip", collectionID)

	// check if file already exists
	_, err = s.client.HeadObject(ctx, &s3_api.HeadObjectInput{
		Bucket: aws.String(s.cfg.S3.Bucket),
		Key:    aws.String(fileName),
	})
	if err == nil {
		return entity.ResultID(fileName), nil
	}

	// Start multipart upload
	var upload *s3_api.CreateMultipartUploadOutput
	upload, err = s.client.CreateMultipartUpload(ctx, &s3_api.CreateMultipartUploadInput{
		Bucket: aws.String(s.cfg.S3.Bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create multipart upload: %w", err)
	}

	defer func() {
		// abort multipart upload on error
		if err != nil {
			_, abortErr := s.client.AbortMultipartUpload(ctx, &s3_api.AbortMultipartUploadInput{
				Bucket:   upload.Bucket,
				Key:      upload.Key,
				UploadId: upload.UploadId,
			})
			if abortErr != nil && !errors.Is(abortErr, context.Canceled) {
				ctxlog.Error(ctx, "failed to abort multipart upload", slog.Any("error", abortErr))
			}
		}
	}()

	// Create buffers for ZIP streaming
	var (
		// Main buffer for ZIP content
		zipBuffer bytes.Buffer
		// Secondary buffer for part uploads
		uploadBuffer bytes.Buffer
		// Create ZIP writer
		zw      = zip.NewWriter(&zipBuffer)
		zipFile io.Writer
	)

	// Create the file inside ZIP archive
	zipFile, err = zw.Create("result.json")
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}

	// Write open json `[`
	_, err = zipFile.Write([]byte("["))
	if err != nil {
		return "", fmt.Errorf("failed to write to zip file: %w", err)
	}

	// Process incoming data
	var (
		completedParts []s3_types.CompletedPart
		partNumber     = int32(1)
	)
	if err = s.processIncomingData(
		ctx, upload, &completedParts, zipFile, &uploadBuffer, &zipBuffer, &partNumber, requests); err != nil {
		return "", err
	}

	// Write close `]` to zip file
	_, err = zipFile.Write([]byte("]"))
	if err != nil {
		return "", fmt.Errorf("failed to write to zip file: %w", err)
	}

	// Close zip writer to finalize ZIP structure
	if err = zw.Close(); err != nil {
		return "", fmt.Errorf("failed to close zip writer: %w", err)
	}

	// Upload any remaining data including ZIP central directory
	if zipBuffer.Len() > 0 {
		uploadBuffer.Reset()
		_, err = io.Copy(&uploadBuffer, bytes.NewReader(zipBuffer.Bytes()))
		if err != nil {
			return "", fmt.Errorf("failed to copy final data to upload buffer: %w", err)
		}

		if err = s.uploadBuffer(ctx, &partNumber, upload, &completedParts, &uploadBuffer); err != nil {
			return "", err
		}
	}

	// Complete multipart upload
	_, err = s.client.CompleteMultipartUpload(ctx, &s3_api.CompleteMultipartUploadInput{
		Bucket:   upload.Bucket,
		Key:      upload.Key,
		UploadId: upload.UploadId,
		MultipartUpload: &s3_types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return entity.ResultID(fileName), nil
}

func (s *Service) processIncomingData(
	ctx context.Context,
	upload *s3_api.CreateMultipartUploadOutput,
	completedParts *[]s3_types.CompletedPart,
	zipFile io.Writer,
	uploadBuffer *bytes.Buffer,
	zipBuffer *bytes.Buffer,
	partNumber *int32,
	requests <-chan entity.RequestChunk,
) error {
	var (
		err       error
		firstPart = true
	)

	for r := range requests {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if r.Err != nil {
			return fmt.Errorf("request error: %w", r.Err)
		}

		// Write array separator `,` if not the first part
		if !firstPart {
			_, err = zipFile.Write([]byte(","))
			if err != nil {
				return fmt.Errorf("failed to write to zip file: %w", err)
			}
		}
		firstPart = false

		// Write data to zip file
		_, err = zipFile.Write(r.Data)
		if err != nil {
			return fmt.Errorf("failed to write to zip file: %w", err)
		}

		// If we have enough data, prepare and upload a part
		if zipBuffer.Len() >= s.cfg.S3.WriteChunkSize {
			// Copy data to upload buffer
			uploadBuffer.Reset()
			_, err = io.Copy(uploadBuffer, bytes.NewReader(zipBuffer.Bytes()))
			if err != nil {
				return fmt.Errorf("failed to copy to upload buffer: %w", err)
			}

			// Upload the part
			if err = s.uploadBuffer(ctx, partNumber, upload, completedParts, uploadBuffer); err != nil {
				return err
			}

			// Clear main buffer after successful upload
			zipBuffer.Reset()
		}
	}

	return nil
}

// uploadBuffer uploads accumulated buffer as a part in multipart upload.
func (s *Service) uploadBuffer(
	ctx context.Context,
	partNumber *int32,
	upload *s3_api.CreateMultipartUploadOutput,
	completedParts *[]s3_types.CompletedPart,
	buf *bytes.Buffer,
) error {
	if buf.Len() == 0 {
		return nil
	}

	fileName := uuid.New().String()

	// Upload the part
	part, err := s.client.UploadPart(ctx, &s3_api.UploadPartInput{
		Bucket:     upload.Bucket,
		Key:        upload.Key,
		UploadId:   upload.UploadId,
		PartNumber: partNumber,
		Body:       bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return fmt.Errorf("failed to upload part %d for file %s: %w", *partNumber, fileName, err)
	}

	// Add the completed part
	*completedParts = append(*completedParts, s3_types.CompletedPart{
		PartNumber: aws.Int32(*partNumber), // clone partNumber
		ETag:       part.ETag,
	})

	// Log successful upload
	ctxlog.Debug(ctx, "part uploaded successfully",
		slog.Int("part_number", int(*partNumber)),
		slog.String("bucket", *upload.Bucket),
		slog.String("key", *upload.Key),
		slog.String("etag", *part.ETag))

	*partNumber++
	return nil
}
