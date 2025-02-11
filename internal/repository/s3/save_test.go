package s3

import (
	"archive/zip"
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestService_SaveResultChan(t *testing.T) {
	s, cfg, ctx := setupTest(t)

	cfg.S3.WriteChunkSize = minPartSize

	const tequestsConunt = 1000

	testData := make([]entity.RequestChunk, 0, tequestsConunt)

	for i := range tequestsConunt {
		testData = append(testData, entity.RequestChunk{
			Data: []byte(`{"id":` + strconv.Itoa(i) + `}`),
		})
	}

	// Create channel and send test data
	requests := make(chan entity.RequestChunk, len(testData))
	for _, chunk := range testData {
		requests <- chunk
	}
	close(requests)

	// Save the result
	resultID, err := s.SaveResultChan(ctx, 123, requests)
	require.NoError(t, err)

	// Verify the saved file
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(string(resultID)),
	})
	require.NoError(t, err)

	// Read and verify the ZIP content
	data, err := io.ReadAll(output.Body)
	require.NoError(t, err)

	// Create a reader for the ZIP data
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	// We should have exactly one file
	require.Len(t, zipReader.File, 1)

	// Read the JSON content from the ZIP
	jsonFile, err := zipReader.File[0].Open()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, jsonFile.Close()) })

	content, err := io.ReadAll(jsonFile)
	require.NoError(t, err)

	// Verify the JSON content
	expectedJSON := "["
	for i := range tequestsConunt {
		expectedJSON += `{"id":` + strconv.Itoa(i) + `}`
		if i < tequestsConunt-1 {
			expectedJSON += ","
		}
	}
	expectedJSON += "]"

	require.JSONEq(t, expectedJSON, string(content))
}
