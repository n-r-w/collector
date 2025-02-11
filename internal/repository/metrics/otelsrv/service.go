package otelsrv

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/n-r-w/ammo-collector/internal/config"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"google.golang.org/grpc/credentials"
)

// Service is a service that provides Otel metrics.
type Service struct {
	name     string
	endpoint string

	collectInterval             time.Duration
	minimumReadMemStatsInterval time.Duration

	clientCrtFile string
	clientKeyFile string
	rootCAFile    string

	kafkaErrors metric.Int64Counter
}

// New creates a new Otel service.
func New(cfg *config.Config) (*Service, error) {
	s := &Service{
		name:                        cfg.App.ServiceName,
		endpoint:                    cfg.OTEL.Endpoint,
		collectInterval:             cfg.OTEL.CollectInterval,
		minimumReadMemStatsInterval: cfg.OTEL.MinimumReadMemStatsInterval,
		clientCrtFile:               cfg.OTEL.ClientCrtFile,
		clientKeyFile:               cfg.OTEL.ClientKeyFile,
		rootCAFile:                  cfg.OTEL.RootCAFile,
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	return s, nil
}

//nolint:lll // ok
func (s *Service) validate() error {
	if s.name == "" {
		return errors.New("name is required")
	}
	if s.endpoint == "" {
		return errors.New("endpoint is required")
	}
	if s.collectInterval <= 0 {
		return errors.New("collect interval must be positive")
	}
	if s.minimumReadMemStatsInterval <= 0 {
		return errors.New("minimum read mem stats interval must be positive")
	}
	if s.clientCrtFile != "" || s.clientKeyFile != "" || s.rootCAFile != "" {
		if s.clientCrtFile == "" || s.clientKeyFile == "" || s.rootCAFile == "" {
			return errors.New("if one of client crt file, client key file, and root CA file is set, then all of them must be set")
		}
	}

	return nil
}

// Register registers the Otel metrics.
func (s *Service) Register(ctx context.Context) (func(ctx context.Context) error, error) {
	otlpGrpcOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(s.endpoint),
	}

	var tlsConf *tls.Config
	if s.clientCrtFile != "" && s.clientKeyFile != "" && s.rootCAFile != "" {
		var err error
		tlsConf, err = getTLS(s.clientCrtFile, s.clientKeyFile, s.rootCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get tls config: %w", err)
		}
	}

	if tlsConf != nil {
		otlpGrpcOptions = append(otlpGrpcOptions, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConf)))
	} else {
		otlpGrpcOptions = append(otlpGrpcOptions, otlpmetricgrpc.WithInsecure())
	}
	exporter, err := otlpmetricgrpc.New(ctx, otlpGrpcOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp exporter: %w", err)
	}

	rc := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(s.name))

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(rc),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(s.collectInterval)),
		),
	)

	otel.SetMeterProvider(mp)

	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(s.minimumReadMemStatsInterval),
		runtime.WithMeterProvider(mp),
	)
	if err != nil {
		_ = mp.Shutdown(ctx)
		return nil, fmt.Errorf("failed to initialize runtime metrics exporting: %w", err)
	}

	if err := s.prepareMetrics(); err != nil {
		_ = mp.Shutdown(ctx)
		return nil, err
	}

	return func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	}, nil
}

// getTLS returns a TLS configuration based on the provided files.
func getTLS(clientCrtFile, clientKeyFile, rootCAFile string) (*tls.Config, error) {
	clientAuth, err := tls.LoadX509KeyPair(clientCrtFile, clientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client key pair: %w", err)
	}

	caCert, err := os.ReadFile(filepath.Clean(rootCAFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read root CA file: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	c := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientAuth},
	}

	return c, nil
}

func (s *Service) prepareMetrics() error {
	var (
		err   error
		meter = otel.GetMeterProvider().Meter(s.name)
	)
	s.kafkaErrors, err = meter.Int64Counter(
		"kafka_errors_total",
		metric.WithDescription("number of kafka errors"),
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka_errors metric: %w", err)
	}

	return nil
}
