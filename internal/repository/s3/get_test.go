package s3

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestService_GetResult(t *testing.T) {
	s, cfg, ctx := setupTest(t)

	cfg.S3.ReadChunkSize = 1 // 1 byte

	const (
		testData = "test data"
		testKey  = "test.txt"
	)

	// Upload test data
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(testKey),
		Body:   strings.NewReader(testData),
	})
	require.NoError(t, err)

	// Get the result
	chunks, err := s.GetResult(ctx, entity.ResultID(testKey))
	require.NoError(t, err)

	var chs []byte
	for chunk := range chunks {
		chs = append(chs, chunk.Data...)
	}

	require.Equal(t, testData, string(chs))
}
