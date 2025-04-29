package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/n-r-w/collector/internal/entity"
	"github.com/n-r-w/collector/internal/pb/api/queue"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/kafkaclient/consumer"
)

// processMessages processes a batch of messages.
func (s *Service) processMessages(ctx context.Context, topic string, partition int32, msgs []consumer.IMessage) error {
	requests := make([]entity.RequestContent, 0, len(msgs))
	for _, msg := range msgs {
		request, err := convertMessageToRequest(msg)
		if err != nil {
			// ignore invalid messages in order to not break the consumer
			if !errors.Is(err, context.Canceled) {
				ctxlog.Error(ctx, "failed to convert message to request",
					slog.Any("error", err),
					slog.Int("batch_size", len(msgs)))

				s.metrics.ObserveKafkaErrors(ctx, err)
			}

			continue
		}
		requests = append(requests, request)
	}

	if len(requests) == 0 {
		return nil
	}

	if err := s.handlers.HandleRequest(ctx, requests); err != nil && !errors.Is(err, context.Canceled) {
		// ignore handling errors in order to not break the consumer
		ctxlog.Error(ctx, "failed to process batch",
			slog.Any("error", err),
			slog.Int("batch_size", len(msgs)))
		s.metrics.ObserveKafkaErrors(ctx, err)
	}
	return nil
}

// convertMessageToRequest converts a Kafka message to RequestContent.
func convertMessageToRequest(msg consumer.IMessage) (entity.RequestContent, error) {
	var req queue.Request
	if err := msg.ReadInProto(&req); err != nil {
		return entity.RequestContent{}, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	content := entity.RequestContent{
		Handler:   req.GetHandler(),
		Headers:   make(map[string][]string, len(req.GetHeaders())),
		Body:      []byte(req.GetBody()),
		CreatedAt: req.GetTimestamp().AsTime(),
	}

	for k, v := range req.GetHeaders() {
		content.Headers[k] = v.GetValues()
	}

	return content, nil
}
