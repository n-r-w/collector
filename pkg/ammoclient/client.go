package ammoclient

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/IBM/sarama"
	queuepb "github.com/n-r-w/collector/internal/pb/api/queue"
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

// WithPassRate sets the percentage of requests that are processed (between 0 and 1).
// Default is 1.0 (all requests are processed).
func WithPassRate(rate float64) Option {
	return func(c *Client) {
		c.passRate = rate
	}
}

// Client represents the ammo client that handles request collection.
type Client struct {
	fn       SendToKafkaFunc
	saramaCh chan<- *sarama.ProducerMessage
	topic    string
	passRate float64
}

// New creates a new ammo client instance.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		passRate: 1.0, // default pass rate is 1.0
	}

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

	if c.passRate < 0 || c.passRate > 1 {
		return nil, errors.New("ammoclient: passRate must be between 0 and 1")
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

	if rand.Float64() > c.passRate { //nolint:gosec // ok for rate
		return nil
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

	return c.sendData(ctx, handler, headersTotal, jsonData)
}

// SendHTTPRequest sends a HTTP request to Kafka using the proper queue message format.
// If headers is nil, it will attempt to extract headers from context using gRPC metadata.
func (c *Client) SendHTTPRequest(
	ctx context.Context, req []byte, handler string, headers map[string][]string,
) error {
	if len(req) == 0 {
		return errors.New("ammoclient: request is empty")
	}
	if handler == "" {
		return errors.New("ammoclient: handler is empty")
	}

	if rand.Float64() > c.passRate { //nolint:gosec // ok for rate
		return nil
	}

	return c.sendData(ctx, handler, headers, req)
}

func (c *Client) sendData(ctx context.Context, handler string, headers map[string][]string, data []byte) error {
	// Create queue message
	queueMsg := &queuepb.Request{
		Handler:   handler,
		Body:      string(data),
		Timestamp: timestamppb.New(time.Now()),
	}

	// Add headers
	queueMsg.Headers = make(map[string]*queuepb.Header, len(headers))
	for k, v := range headers {
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
