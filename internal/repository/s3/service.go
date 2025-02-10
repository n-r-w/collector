package s3

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3_api "github.com/aws/aws-sdk-go-v2/service/s3"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/usecases/apiprocessor"
	"github.com/n-r-w/ammo-collector/internal/usecases/finalizer"
	"github.com/n-r-w/bootstrap"
)

// Service implements S3 repository.
type Service struct {
	client *s3_api.Client
	cfg    *config.Config
}

var (
	_ bootstrap.IService         = (*Service)(nil)
	_ finalizer.IResultChanSaver = (*Service)(nil)
	_ apiprocessor.IResultGetter = (*Service)(nil)
)

// New creates a new instance of the S3 service.
func New(cfg *config.Config) (*Service, error) {
	const minPartSize = 5 << 20 // 5MB

	if cfg.S3.WriteChunkSize < minPartSize {
		return nil, fmt.Errorf("write chunk size is too small: %d, min part size: %d", cfg.S3.WriteChunkSize, minPartSize)
	}

	return &Service{
		cfg: cfg,
	}, nil
}

// Info returns information about the service. Implements bootstrap.IService Info method.
func (s *Service) Info() bootstrap.Info {
	return bootstrap.Info{
		Name: "S3",
	}
}

type resolver struct {
	endpoint     *url.URL
	minioSupport bool
}

func (r *resolver) ResolveEndpoint(
	ctx context.Context, params s3_api.EndpointParameters,
) (endpoint smithyendpoints.Endpoint, err error) {
	if r.minioSupport {
		if params.Bucket == nil {
			return smithyendpoints.Endpoint{}, errors.New("bucket is required")
		}

		u := *r.endpoint
		u.Path += "/" + *params.Bucket
		return smithyendpoints.Endpoint{URI: u}, nil
	}

	return smithyendpoints.Endpoint{URI: *r.endpoint}, nil
}

// Start starts the service. Implements bootstrap.IService Start method.
func (s *Service) Start(ctx context.Context) error {
	awsDefaultConfig, err := aws_config.LoadDefaultConfig(ctx,
		aws_config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s.cfg.S3.AccessKey, s.cfg.S3.SecretKey, "")),
	)
	if err != nil {
		return fmt.Errorf("failed to load default config: %w", err)
	}

	uri, err := url.Parse(s.cfg.S3.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint: %w", err)
	}

	s.client = s3_api.NewFromConfig(awsDefaultConfig, func(o *s3_api.Options) {
		o.Region = s.cfg.S3.Region
		o.EndpointResolverV2 = &resolver{endpoint: uri, minioSupport: s.cfg.S3.MinioSupport}
		o.UsePathStyle = s.cfg.S3.UsePathStyle
	})

	return nil
}

// Stop stops the service. Implements bootstrap.IService Stop method.
func (s *Service) Stop(_ context.Context) error {
	return nil
}
