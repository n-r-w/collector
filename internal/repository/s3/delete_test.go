package s3

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n-r-w/ammo-collector/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestService_CleanObjectStorage(t *testing.T) {
	s, cfg, ctx := setupTest(t)

	tests := []struct {
		name      string
		resultIDs []entity.ResultID
		err       bool
	}{
		{
			name:      "successful deletion",
			resultIDs: []entity.ResultID{"test1", "test2"},
			err:       false,
		},
		{
			name:      "empty result IDs",
			resultIDs: []entity.ResultID{},
			err:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add test objects
			for _, id := range tt.resultIDs {
				_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: aws.String(cfg.S3.Bucket),
					Key:    aws.String(string(id)),
					Body:   strings.NewReader("test data"),
				})
				require.NoError(t, err)
			}

			err := s.CleanObjectStorage(ctx, tt.resultIDs)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
