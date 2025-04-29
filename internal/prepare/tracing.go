package prepare

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n-r-w/collector/internal/config"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/credentials"
)

const (
	// ZipkinExporterType is the type of the zipkin exporter.
	ZipkinExporterType = "ZIPKIN"
	// OtlpGrpcExporterType is the type of the OTLP gRPC exporter.
	OtlpGrpcExporterType = "OTLP-GRPC"
	// OtlpHTTPExporterType is the type of the OTLP HTTP exporter.
	OtlpHTTPExporterType = "OTLP-HTTP"
)

// InitTracer initializes OpenTelemetry tracer.
func InitTracer(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	if !cfg.Tracing.Enabled {
		return nil, nil //nolint:nilnil // ok
	}

	if err := validateTracingConfig(cfg); err != nil {
		return nil, err
	}

	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracing exporter: %w", err)
	}

	tp := createTracerProvider(cfg, exporter)
	otel.SetTracerProvider(tp)

	cleanup := func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
		return nil
	}

	if err := setupPropagators(cfg); err != nil {
		return nil, err
	}

	return cleanup, nil
}

// validateTracingConfig validates the tracing configuration.
func validateTracingConfig(cfg *config.Config) error {
	if cfg.Tracing.Endpoint == "" {
		return errors.New("environment variable TRACING_ENDPOINT must be set")
	}
	if len(cfg.Tracing.Propagators) == 0 {
		return errors.New("environment variable TRACING_PROPAGATORS must be set")
	}
	if cfg.Tracing.Exporter == "" {
		return errors.New("environment variable TRACING_EXPORTER must be set")
	}
	//nolint:lll // ok
	if cfg.Tracing.ClientCrtFile != "" || cfg.Tracing.ClientKeyFile != "" || cfg.Tracing.RootCAFile != "" {
		if strings.EqualFold(cfg.Tracing.Exporter, ZipkinExporterType) {
			return errors.New("environment variables TRACING_CLIENT_CRT_FILE, TRACING_CLIENT_KEY_FILE, and TRACING_ROOT_CA_FILE are not supported for Zipkin exporter")
		}

		if cfg.Tracing.ClientCrtFile == "" || cfg.Tracing.ClientKeyFile == "" || cfg.Tracing.RootCAFile == "" {
			return errors.New("if one of environment variables TRACING_CLIENT_CRT_FILE, TRACING_CLIENT_KEY_FILE, and TRACING_ROOT_CA_FILE is set, then all of them must be set")
		}
	}
	return nil
}

// createExporter creates a span exporter based on the configuration.
func createExporter(ctx context.Context, cfg *config.Config) (sdktrace.SpanExporter, error) {
	switch strings.ToUpper(cfg.Tracing.Exporter) {
	case OtlpGrpcExporterType:
		return createOTLPGRPCExporter(ctx, cfg)
	case ZipkinExporterType:
		return zipkin.New(cfg.Tracing.Endpoint)
	case OtlpHTTPExporterType:
		return createOTLPHTTPExporter(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown tracing exporter: %s", cfg.Tracing.Exporter)
	}
}

// createOTLPGRPCExporter creates an OTLP gRPC exporter.
func createOTLPGRPCExporter(ctx context.Context, cfg *config.Config) (sdktrace.SpanExporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpointURL(cfg.Tracing.Endpoint),
		otlptracegrpc.WithTimeout(cfg.Tracing.Timeout),
	}

	options = appendTLSOptions(options, cfg, func(tlsConf *tls.Config) otlptracegrpc.Option {
		return otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConf))
	}, otlptracegrpc.WithInsecure())

	return otlptracegrpc.New(ctx, options...)
}

// createOTLPHTTPExporter creates an OTLP HTTP exporter.
func createOTLPHTTPExporter(ctx context.Context, cfg *config.Config) (sdktrace.SpanExporter, error) {
	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Tracing.Endpoint),
		otlptracehttp.WithTimeout(cfg.Tracing.Timeout),
	}

	options = appendTLSOptions(options, cfg, otlptracehttp.WithTLSClientConfig, otlptracehttp.WithInsecure())

	return otlptracehttp.New(ctx, options...)
}

// appendTLSOptions appends TLS options to the given options slice based on the configuration.
func appendTLSOptions[T any](options []T, cfg *config.Config, withTLS func(*tls.Config) T, withInsecure T) []T {
	if cfg.Tracing.ClientCrtFile != "" && cfg.Tracing.ClientKeyFile != "" && cfg.Tracing.RootCAFile != "" {
		if tlsConf, err := initTracerTLS(
			cfg.Tracing.ClientCrtFile,
			cfg.Tracing.ClientKeyFile,
			cfg.Tracing.RootCAFile); err == nil {
			options = append(options, withTLS(tlsConf))
		}
	} else {
		options = append(options, withInsecure)
	}
	return options
}

// createTracerProvider creates a new tracer provider with the given exporter.
func createTracerProvider(cfg *config.Config, exporter sdktrace.SpanExporter) *sdktrace.TracerProvider {
	attrs := []attribute.KeyValue{semconv.ServiceNameKey.String(cfg.App.ServiceName)}
	rc := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(rc),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Tracing.SamplingRate))),
	)
}

// setupPropagators configures the global propagators based on the configuration.
func setupPropagators(cfg *config.Config) error {
	propagators := []propagation.TextMapPropagator{
		propagation.TraceContext{},
		propagation.Baggage{},
	}

	for _, p := range cfg.Tracing.Propagators {
		switch strings.ToUpper(p) {
		case "B3":
			propagators = append(propagators, b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader|b3.B3SingleHeader)))
		case "JAEGER":
			propagators = append(propagators, jaeger.Jaeger{})
		default:
			return fmt.Errorf("unknown trace propagator: %s", p)
		}
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))
	return nil
}

// initTracerTLS initializes TLS configuration for the OpenTelemetry tracer.
func initTracerTLS(clientCrtFile, clientKeyFile, rootCAFile string) (*tls.Config, error) {
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
