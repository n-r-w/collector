package ammoclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	queuepb "github.com/n-r-w/ammo-collector/internal/pb/api/queue"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SendToKafkaFunc is a function that sends a message to Kafka.
type SendToKafkaFunc func(ctx context.Context, data []byte) error

// Option is a function that configures the ammo client.
type Option func(*Client)

// WithSendToKafka configures the ammo client to send messages to Kafka.
func WithSendToKafka(fn SendToKafkaFunc) Option {
	return func(c *Client) {
		c.fn = fn
	}
}

// WithSendToKafkaSarama configures the ammo client to send messages to Kafka using sarama.
func WithSendToKafkaSarama(ch chan<- *sarama.ProducerMessage, topic string) Option {
	return func(c *Client) {
		c.saramaCh = ch
		c.topic = topic
	}
}

// Client represents the ammo client that handles request collection.
type Client struct {
	fn       SendToKafkaFunc
	saramaCh chan<- *sarama.ProducerMessage
	topic    string
}

// New creates a new ammo client instance.
func New(opts ...Option) (*Client, error) {
	c := &Client{}

	for _, opt := range opts {
		opt(c)
	}

	if c.fn == nil && c.saramaCh == nil {
		return nil, errors.New("ammoclient: neither SendToKafka nor SendToKafkaSarama is set")
	}

	if c.fn != nil && c.saramaCh != nil {
		return nil, errors.New("ammoclient: both SendToKafka and SendToKafkaSarama are set")
	}

	if c.saramaCh != nil && c.topic == "" {
		return nil, errors.New("ammoclient: SendToKafkaSarama is set, but topic is empty")
	}

	return c, nil
}

// SendGRPCRequest sends a gRPC request to Kafka using the proper queue message format.
// If headers is nil, it will attempt to extract headers from context using gRPC metadata.
func (c *Client) SendGRPCRequest(
	ctx context.Context, req proto.Message, handler string, headers map[string][]string,
) error {
	if req == nil {
		return errors.New("ammoclient: request is nil")
	}
	if handler == "" {
		return errors.New("ammoclient: handler is empty")
	}

	headersTotal := make(map[string][]string, len(headers))
	for k, v := range headers {
		headersTotal[k] = v
	}

	// Extract additional headers from context if not provided
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range md {
			if headersTotal[k] != nil {
				continue
			}
			headersTotal[k] = v
		}
	}

	// Convert protobuf message to JSON
	jsonData, err := protojson.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request to JSON: %w", err)
	}

	// Create queue message
	queueMsg := &queuepb.Request{
		Handler:   handler,
		Body:      string(jsonData),
		Timestamp: timestamppb.New(time.Now()),
	}

	// Add headers
	queueMsg.Headers = make(map[string]*queuepb.Header, len(headersTotal))
	for k, v := range headersTotal {
		queueMsg.Headers[k] = &queuepb.Header{
			Values: v,
		}
	}

	// Marshal queue message
	msgData, err := proto.Marshal(queueMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message: %w", err)
	}

	// Send message
	if c.fn != nil {
		if err := c.fn(ctx, msgData); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		return nil
	} else {
		msg := &sarama.ProducerMessage{
			Topic: c.topic,
			Value: sarama.ByteEncoder(msgData),
		}

		select {
		case c.saramaCh <- msg:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
