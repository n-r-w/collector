package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/collector/internal/config"
	"github.com/n-r-w/collector/internal/telemetry"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/kafkaclient/consumer"
)

// Service is a Kafka consumer implementation.
type Service struct {
	handlers IHandlers
	metrics  telemetry.IMetrics
	client   *consumer.Consumer
}

// New creates a new Kafka consumer implementation.
func New(ctx context.Context, cfg *config.Config, handlers IHandlers, metrics telemetry.IMetrics) (*Service, error) {
	service := &Service{
		handlers: handlers,
		metrics:  metrics,
	}

	if err := service.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create Kafka batch consumer
	client, err := consumer.NewSimple(
		ctx,
		cfg.App.ServiceName,
		cfg.Kafka.KafkaBrokers,
		cfg.Kafka.GroupID,
		cfg.App.ServiceName,
		service.processMessages,
		[]string{cfg.Kafka.KafkaTopic},
		consumer.WithBatchSize(cfg.Kafka.BatchSize),
		consumer.WithFlushTimeout(cfg.Kafka.FlushTimeout),
		consumer.WithBatchTopics(cfg.Kafka.KafkaTopic),
		consumer.WithErrorLogger(service),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	service.client = client

	return service, nil
}

// Validate validates the Kafka consumer service configuration.
func (s *Service) Validate(cfg *config.Config) error {
	var errs []error

	if len(cfg.Kafka.KafkaBrokers) == 0 {
		errs = append(errs, errors.New("brokers list cannot be empty"))
	}

	if cfg.Kafka.KafkaTopic == "" {
		errs = append(errs, errors.New("topic cannot be empty"))
	}

	if cfg.Kafka.GroupID == "" {
		errs = append(errs, errors.New("group ID cannot be empty"))
	}

	if cfg.Kafka.BatchSize <= 0 {
		errs = append(errs, errors.New("batch size must be positive"))
	}

	if cfg.Kafka.FlushTimeout <= 0 {
		errs = append(errs, errors.New("flush timeout must be positive"))
	}

	return errors.Join(errs...)
}

// LogError implements the consumer.ErrorLogger interface.
func (s *Service) LogError(ctx context.Context, err error) {
	ctxlog.Error(ctx, "kafka consumer error", slog.Any("error", err))
}

// Info implements the bootstrap.IService Info method.
func (s *Service) Info() bootstrap.Info {
	return bootstrap.Info{
		Name: "Kafka Consumer",
	}
}

// Start starts the service. Implements bootstrap.IService Start method.
func (s *Service) Start(ctx context.Context) error {
	// Start Kafka consumer
	if err := s.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Kafka consumer: %w", err)
	}

	return nil
}

// Stop stops the service. Implements bootstrap.IService Stop method.
func (s *Service) Stop(ctx context.Context) error {
	// Stop Kafka consumer
	if err := s.client.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop Kafka consumer: %w", err)
	}

	return nil
}
