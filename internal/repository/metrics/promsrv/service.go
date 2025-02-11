package promsrv

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Service is a service that provides Prometheus metrics.
type Service struct {
	kafkaErrors prometheus.Counter
}

// New creates a new Prometheus service.
func New() *Service {
	return &Service{}
}

// Register registers the Prometheus metrics.
// Returns cleanup function.
func (s *Service) Register(_ context.Context) (func(ctx context.Context) error, error) {
	s.prepareMetrics()
	return func(_ context.Context) error { return nil }, nil
}

func (s *Service) prepareMetrics() {
	s.kafkaErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_errors_total",
			Help: "Number of kafka errors",
		})
}
