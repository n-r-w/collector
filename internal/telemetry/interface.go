package telemetry

import "context"

// IMetrics represents metrics interface.
type IMetrics interface {
	// ObserveKafkaErrors observe kafka errors.
	ObserveKafkaErrors(ctx context.Context, err error)
}
