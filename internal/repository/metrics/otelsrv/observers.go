package otelsrv

import "context"

// ObserveKafkaErrors observe kafka errors.
func (s *Service) ObserveKafkaErrors(ctx context.Context, err error) {
	if err == nil {
		return
	}

	s.kafkaErrors.Add(ctx, 1)
}
