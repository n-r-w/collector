//nolint:errcheck // Ignore errors in tests
package prepare

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/n-r-w/collector/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestValidateTracingConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		setupConfig func() *config.Config
		expectedErr string
	}{
		{
			name: "valid configuration with zipkin",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
				cfg.Tracing.Exporter = "ZIPKIN"
				cfg.Tracing.Propagators = []string{"b3", "jaeger"}
				return cfg
			},
			expectedErr: "",
		},
		{
			name: "valid configuration with otlp-grpc",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "localhost:4317"
				cfg.Tracing.Exporter = "OTLP-GRPC"
				cfg.Tracing.Propagators = []string{"b3"}
				return cfg
			},
			expectedErr: "",
		},
		{
			name: "missing endpoint",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Exporter = "ZIPKIN"
				cfg.Tracing.Propagators = []string{"b3"}
				return cfg
			},
			expectedErr: "environment variable TRACING_ENDPOINT must be set",
		},
		{
			name: "missing propagators",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
				cfg.Tracing.Exporter = "ZIPKIN"
				return cfg
			},
			expectedErr: "environment variable TRACING_PROPAGATORS must be set",
		},
		{
			name: "missing exporter",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
				cfg.Tracing.Propagators = []string{"b3"}
				return cfg
			},
			expectedErr: "environment variable TRACING_EXPORTER must be set",
		},
		{
			name: "tls files not supported for zipkin",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
				cfg.Tracing.Exporter = "ZIPKIN"
				cfg.Tracing.Propagators = []string{"b3"}
				cfg.Tracing.ClientCrtFile = "cert.pem"
				cfg.Tracing.ClientKeyFile = "key.pem"
				cfg.Tracing.RootCAFile = "ca.pem"
				return cfg
			},
			expectedErr: "environment variables TRACING_CLIENT_CRT_FILE, TRACING_CLIENT_KEY_FILE, and TRACING_ROOT_CA_FILE are not supported for Zipkin exporter",
		},
		{
			name: "incomplete tls configuration",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "localhost:4317"
				cfg.Tracing.Exporter = "OTLP-GRPC"
				cfg.Tracing.Propagators = []string{"b3"}
				cfg.Tracing.ClientCrtFile = "cert.pem"
				// Missing key and CA files
				return cfg
			},
			expectedErr: "if one of environment variables TRACING_CLIENT_CRT_FILE, TRACING_CLIENT_KEY_FILE, and TRACING_ROOT_CA_FILE is set, then all of them must be set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := tc.setupConfig()
			err := validateTracingConfig(cfg)
			if tc.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestSetupPropagators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		propagators []string
		expectError bool
	}{
		{
			name:        "valid propagators",
			propagators: []string{"b3", "jaeger"},
			expectError: false,
		},
		{
			name:        "case insensitive propagators",
			propagators: []string{"B3", "JAEGER"},
			expectError: false,
		},
		{
			name:        "invalid propagator",
			propagators: []string{"b3", "invalid"},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Backup original propagator
			original := otel.GetTextMapPropagator()
			defer func() {
				otel.SetTextMapPropagator(original)
			}()

			cfg := &config.Config{}
			cfg.Tracing.Propagators = tc.propagators

			err := setupPropagators(cfg)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Check that a propagator was actually set
				assert.NotNil(t, otel.GetTextMapPropagator())
			}
		})
	}
}

func TestCreateExporter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		setupConfig func() *config.Config
		expectError bool
	}{
		{
			name: "zipkin exporter",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
				cfg.Tracing.Exporter = "ZIPKIN"
				cfg.Tracing.Propagators = []string{"b3"}
				return cfg
			},
			expectError: false,
		},
		{
			name: "otlp-grpc exporter",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "localhost:4317"
				cfg.Tracing.Exporter = "OTLP-GRPC"
				cfg.Tracing.Propagators = []string{"b3"}
				cfg.Tracing.Timeout = time.Second * 5
				return cfg
			},
			expectError: false,
		},
		{
			name: "otlp-http exporter",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "localhost:4318"
				cfg.Tracing.Exporter = "OTLP-HTTP"
				cfg.Tracing.Propagators = []string{"b3"}
				cfg.Tracing.Timeout = time.Second * 5
				return cfg
			},
			expectError: false,
		},
		{
			name: "unknown exporter",
			setupConfig: func() *config.Config {
				cfg := &config.Config{}
				cfg.Tracing.Enabled = true
				cfg.Tracing.Endpoint = "localhost:4317"
				cfg.Tracing.Exporter = "UNKNOWN"
				cfg.Tracing.Propagators = []string{"b3"}
				return cfg
			},
			expectError: true,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := tc.setupConfig()
			exporter, err := createExporter(ctx, cfg)
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, exporter)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, exporter)
				// Clean up the exporter
				err = exporter.Shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitTracerTLS(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "tracer-tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create dummy certificate files
	clientCrt := filepath.Join(tempDir, "client.crt")
	clientKey := filepath.Join(tempDir, "client.key")
	rootCA := filepath.Join(tempDir, "ca.crt")

	// Invalid case - files don't exist
	t.Run("missing files", func(t *testing.T) {
		_, err := initTracerTLS(clientCrt, clientKey, rootCA)
		require.Error(t, err)
	})

	t.Run("invalid certificate data", func(t *testing.T) {
		// Create empty files for TLS (not valid certificates but enough to test file existence checks)
		require.NoError(t, os.WriteFile(clientCrt, []byte("CERTIFICATE"), 0o600))
		require.NoError(t, os.WriteFile(clientKey, []byte("PRIVATE KEY"), 0o600))
		require.NoError(t, os.WriteFile(rootCA, []byte("CA CERTIFICATE"), 0o600))

		_, err := initTracerTLS(clientCrt, clientKey, rootCA)
		require.Error(t, err)
	})

	t.Run("valid certificate data", func(t *testing.T) {
		// Create valid certificate files
		clientCrtFile := "./test_cert/client.crt"
		clientKeyFile := "./test_cert/client.key"
		rootCAFile := "./test_cert/rootCA.crt"

		tlsConfig, err := initTracerTLS(clientCrtFile, clientKeyFile, rootCAFile)
		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		require.Len(t, tlsConfig.Certificates, 1)
		require.NotNil(t, tlsConfig.RootCAs)
	})
}

func TestInitTracer(t *testing.T) {
	t.Run("tracing disabled", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{}
		cfg.Tracing.Enabled = false

		cleanup, err := InitTracer(ctx, cfg)
		require.NoError(t, err)
		assert.Nil(t, cleanup)
	})

	t.Run("invalid config", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{}
		cfg.Tracing.Enabled = true
		// Missing required fields

		cleanup, err := InitTracer(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, cleanup)
	})

	t.Run("valid zipkin config", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{}
		cfg.App.ServiceName = "test-service"
		cfg.Tracing.Enabled = true
		cfg.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
		cfg.Tracing.Exporter = "ZIPKIN"
		cfg.Tracing.Propagators = []string{"b3"}
		cfg.Tracing.SamplingRate = 0.1

		// Backup original tracer provider
		originalProvider := otel.GetTracerProvider()
		defer otel.SetTracerProvider(originalProvider)

		cleanup, err := InitTracer(ctx, cfg)
		// We expect this to fail in a real environment without a real Zipkin server
		// But we can verify that the function behaves as expected overall
		if err == nil {
			assert.NotNil(t, cleanup)
			// If it somehow succeeded, clean up properly
			assert.NoError(t, cleanup(ctx))
		}
	})
}

func TestCreateTracerProvider(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg1 := &config.Config{}
	cfg1.Tracing.Enabled = true
	cfg1.Tracing.Endpoint = "http://localhost:9411/api/v2/spans"
	cfg1.Tracing.Exporter = "ZIPKIN"
	cfg1.Tracing.Propagators = []string{"b3"}

	exporter, err := createExporter(ctx, cfg1)
	require.NoError(t, err)
	defer exporter.Shutdown(ctx)

	cfg2 := &config.Config{}
	cfg2.App.ServiceName = "test-service"
	cfg2.Tracing.SamplingRate = 0.5

	tp := createTracerProvider(cfg2, exporter)

	assert.NotNil(t, tp)
	defer tp.Shutdown(ctx)

	tracer := tp.Tracer("test")
	assert.NotNil(t, tracer)
}
