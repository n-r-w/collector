package s3

import (
	"context"
	"testing"

	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/ctxlog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

const (
	testBucket    = "miniobucket"
	testAccessKey = "minioaccesskey"
	testSecretKey = "miniosecretkey"
)

func setupTest(t *testing.T) (*Service, *config.Config, context.Context) {
	t.Helper()

	ctx := ctxlog.MustContext(context.Background(), ctxlog.WithTesting(t))

	minioContainer, err := minio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z",
		// Create default bucket
		testcontainers.WithStartupCommand(testcontainers.NewRawCommand([]string{"mkdir", "/data/" + testBucket})),
		testcontainers.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     testAccessKey,
			"MINIO_ROOT_PASSWORD": testSecretKey,
		}),
	)
	require.NoError(t, err)
	require.NoError(t, minioContainer.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, minioContainer.Terminate(ctx))
	})
	url, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err)

	url = "http://" + url

	cfg := &config.Config{}
	cfg.S3.ReadChunkSize = 52428800
	cfg.S3.WriteChunkSize = 52428800
	cfg.S3.Endpoint = url
	cfg.S3.Bucket = testBucket
	cfg.S3.AccessKey = testAccessKey
	cfg.S3.SecretKey = testSecretKey
	cfg.S3.MinioSupport = true
	cfg.S3.UsePathStyle = true

	s, err := New(cfg)
	require.NoError(t, err)

	require.NoError(t, s.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, s.Stop(ctx))
	})

	return s, cfg, ctx
}
