package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/n-r-w/ctxlog"
)

// Config holds all configuration for the service.
type Config struct {
	App struct {
		ServiceName    string `env:"SERVICE_NAME" envDefault:"ammo-collector"`
		LogLevelString string `env:"LOG_LEVEL" envDefault:"INFO"` // DEBUG, INFO, WARN, ERROR
		LogLevel       slog.Level
		EnvTypeString  string `env:"ENV_TYPE" envDefault:"DEV"` // PROD, DEV
		EnvType        ctxlog.EnvType
	}

	// Server configuration.
	Server struct {
		// Host is the server host.
		Host string `env:"SERVER_HOST" envDefault:"localhost"`
		// ShutdownTimeout is the timeout for graceful shutdown.
		ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
		// RecoverPanics enables panic recovery middleware.
		RecoverPanics bool `env:"RECOVER_PANICS" envDefault:"true"`
		// ProfilingEndpoint is the endpoint for pprof profiling.
		ProfilingEndpoint string `env:"PROFILING_ENDPOINT"`
		// MetricsType defines the type of metrics to use (PROMETHEUS or OTEL).
		MetricsType string `env:"METRICS_TYPE" envDefault:"PROMETHEUS"` // PROMETHEUS, OTEL
	}

	GRPC struct {
		// GRPCPort is the gRPC server port.
		Port int `env:"GRPC_PORT" envDefault:"8090"`
	}

	// HTTP server configuration.
	HTTP struct {
		// HTTPPort is the HTTP server port.
		Port int `env:"HTTP_PORT" envDefault:"8091"`
		// ReadHeaderTimeout is the amount of time allowed to read request headers.
		ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"10s"`
	}

	// Prometheus metrics configuration.
	Prometheus struct {
		Endpoint string `env:"METRICS_PROMETHEUS_ENDPOINT" envDefault:"localhost:8092"`
	}

	// OTEL metrics configuration.
	OTEL struct {
		Endpoint                    string        `env:"METRICS_OTEL_ENDPOINT" envDefault:"localhost:4317"`
		CollectInterval             time.Duration `env:"METRICS_OTEL_COLLECT_INTERVAL" envDefault:"60s"`
		MinimumReadMemStatsInterval time.Duration `env:"METRICS_OTEL_MINIMUM_READ_MEM_STATS_INTERVAL" envDefault:"15s"`
		ClientCrtFile               string        `env:"METRICS_OTEL_CLIENT_CERT_FILE"`
		ClientKeyFile               string        `env:"METRICS_OTEL_CLIENT_KEY_FILE"`
		RootCAFile                  string        `env:"METRICS_OTEL_ROOT_CA_FILE"`
	}

	// Tracing configuration.
	Tracing struct {
		Enabled      bool          `env:"TRACING_ENABLED" envDefault:"false"`
		Endpoint     string        `env:"TRACING_ENDPOINT"`
		SamplingRate float64       `env:"TRACING_SAMPLING_RATE" envDefault:"1.0"`
		Timeout      time.Duration `env:"TRACING_TIMEOUT" envDefault:"10s"`
		// Exporter for OpenTelemetry.
		Exporter    string   `env:"TRACING_EXPORTER"`    // OTLP-GRPC, OTLP-HTTP, ZIPKIN
		Propagators []string `env:"TRACING_PROPAGATORS"` // B3, JAEGER
		// For TLS set all three fields. Not supported for Zipkin exporter.
		ClientCrtFile string `env:"TRACING_CLIENT_CERT_FILE"`
		ClientKeyFile string `env:"TRACING_CLIENT_KEY_FILE"`
		RootCAFile    string `env:"TRACING_ROOT_CA_FILE"`
	}

	// Kafka configuration.
	Kafka struct {
		KafkaBrokers []string      `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
		KafkaTopic   string        `env:"KAFKA_TOPIC" envDefault:"ammo-collector"`
		GroupID      string        `env:"KAFKA_GROUP_ID" envDefault:"ammo-collector"`
		BatchSize    int           `env:"KAFKA_BATCH_SIZE" envDefault:"100"`
		FlushTimeout time.Duration `env:"KAFKA_FLUSH_TIMEOUT" envDefault:"10s"`
	}

	// Storage configuration.
	Database struct {
		DSN string `env:"DATABASE_URL,required"`
	}

	// S3 configuration.
	S3 struct {
		Endpoint     string `env:"S3_ENDPOINT,required"`
		Region       string `env:"S3_REGION,required"`
		AccessKey    string `env:"S3_ACCESS_KEY,required"`
		SecretKey    string `env:"S3_SECRET_KEY,required"`
		Bucket       string `env:"S3_BUCKET,required"`
		MinioSupport bool   `env:"S3_MINIO_SUPPORT" envDefault:"true"`
		UsePathStyle bool   `env:"S3_USE_PATH_STYLE" envDefault:"true"`
		// ReadChunkSize is the size in bytes of the chunk to read from S3.
		ReadChunkSize int `env:"S3_READ_CHUNK_SIZE" envDefault:"5242880"`
		// WriteChunkSize is the size in bytes of the chunk to write to S3.
		WriteChunkSize int `env:"S3_WRITE_CHUNK_SIZE" envDefault:"52428800"` // 50MB
	}

	// Collection configuration.
	Collection struct {
		// CacheUpdateInterval is the interval for updating the collection cache.
		CacheUpdateInterval time.Duration `env:"CACHE_UPDATE_INTERVAL" envDefault:"10s"`
		// CacheUpdateIntervalJitter is the jitter for the cache update interval.
		CacheUpdateIntervalJitter time.Duration `env:"CACHE_UPDATE_INTERVAL_JITTER" envDefault:"1s"`
		// CleanupInterval is the interval for cleaning up old collections.
		CleanupInterval time.Duration `env:"CLEANUP_INTERVAL" envDefault:"1h"`
		// CleanupIntervalJitter is the jitter for the cleanup interval.
		CleanupIntervalJitter time.Duration `env:"CLEANUP_INTERVAL_JITTER" envDefault:"1m"`
		// RetentionPeriod is the duration for which collections are retained.
		RetentionPeriod time.Duration `env:"RETENTION_PERIOD" envDefault:"168h"` // 7 days
		// FinalizerInterval is the interval for checking collection status.
		FinalizerInterval time.Duration `env:"FINALIZER_INTERVAL" envDefault:"10s"`
		// FinalizerIntervalJitter is the jitter for the finalizer interval.
		FinalizerIntervalJitter time.Duration `env:"FINALIZER_INTERVAL_JITTER" envDefault:"1s"`
		// FinalizerConcurrency is the concurrency for finalizing collections.
		FinalizerConcurrency int `env:"FINALIZER_CONCURRENCY" envDefault:"10"`
		// FinalizerMaxCollections is the maximum number of collections to finalize per interval.
		FinalizerMaxCollections int `env:"FINALIZER_MAX_COLLECTIONS" envDefault:"10"`
		// FinalizerResultBatchSize is the batch size for finalizing collections.
		FinalizerResultBatchSize int `env:"FINALIZER_RESULT_BATCH_SIZE" envDefault:"100"`
		// MaxRequestsPerCollection is the maximum number of requests per collection.
		MaxRequestsPerCollection int `env:"MAX_REQUESTS_PER_COLLECTION" envDefault:"10000"`
	}
}

// MustNew creates a new Config instance from environment variables.
func MustNew() *Config {
	var err error

	_ = godotenv.Load()

	cfg := &Config{}
	if err = env.ParseWithOptions(cfg, env.Options{Prefix: "AMMO_COLLECTOR_"}); err != nil {
		panic(fmt.Errorf("failed to parse config: %w", err))
	}

	if err = cfg.App.LogLevel.UnmarshalText([]byte(cfg.App.LogLevelString)); err != nil {
		panic(fmt.Errorf("invalid log level %s: %w", cfg.App.LogLevelString, err))
	}

	if cfg.App.EnvType, err = ctxlog.EnvTypeFromString(cfg.App.EnvTypeString); err != nil {
		panic(fmt.Errorf("invalid env type %s: %w", cfg.App.EnvTypeString, err))
	}

	return cfg
}
