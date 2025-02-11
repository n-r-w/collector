package promsrv

import "context"

// ObserveKafkaErrors observe kafka errors.
func (s *Service) ObserveKafkaErrors(_ context.Context, err error) {
	if err == nil {
		return
	}

	s.kafkaErrors.Inc()
}
