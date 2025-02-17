package clienttest

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	queuepb "github.com/n-r-w/ammo-collector/internal/pb/api/queue"
	"github.com/n-r-w/ammo-collector/pkg/ammoclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestGRPC(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const (
		messageData = "test message"
		handlerData = "test heandler"
	)

	headersData := map[string][]string{
		"Content-Type": {"application/json"},
		"X-Test-Header": {
			"test header value 1",
			"test header value 2",
		},
	}

	metaData := map[string]string{
		"meta-header":  "meta-value",
		"another-meta": "value1",
	}
	ctx = metadata.NewIncomingContext(ctx, metadata.New(metaData))

	checkFunc := func(ctx context.Context, data []byte) error {
		req := &queuepb.Request{}
		require.NoError(t, proto.Unmarshal(data, req))

		msg := &TestMessage{}
		require.NoError(t, protojson.Unmarshal([]byte(req.GetBody()), msg))

		require.Equal(t, messageData, msg.GetMessage())
		require.Equal(t, handlerData, req.GetHandler())

		// Compare headers
		// All headers from headersData and metaData should be present
		for k, v := range headersData {
			header, ok := req.GetHeaders()[k]
			require.True(t, ok, "Header %s not found", k)
			require.ElementsMatch(t, v, header.GetValues())
		}

		for k, v := range metaData {
			meta, ok := req.GetHeaders()[k]
			require.True(t, ok, "Header %s not found", k)
			require.ElementsMatch(t, []string{v}, meta.GetValues())
		}

		return nil
	}

	// WithSendToKafka
	c, err := ammoclient.New(ammoclient.WithSendToKafka(checkFunc))
	require.NoError(t, err)

	require.NoError(t, c.SendGRPCRequest(
		ctx, &TestMessage{Message: messageData}, handlerData, headersData))

	// WithSendToKafkaSarama
	testCh := make(chan *sarama.ProducerMessage, 1)

	c, err = ammoclient.New(ammoclient.WithSendToKafkaSarama(testCh, "test-topic"))
	require.NoError(t, err)

	require.NoError(t, c.SendGRPCRequest(
		ctx, &TestMessage{Message: messageData}, handlerData, headersData))

	msg := <-testCh
	require.NotNil(t, msg)
	require.Equal(t, "test-topic", msg.Topic)
	data, err := msg.Value.Encode()
	require.NoError(t, err)
	require.NoError(t, checkFunc(ctx, data))
}

func TestHTTP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const (
		messageData = "test http message"
		handlerData = "test http handler"
	)

	headersData := map[string][]string{
		"Content-Type": {"application/json"},
		"X-Test-Header": {
			"test header value 1",
			"test header value 2",
		},
	}

	checkFunc := func(ctx context.Context, data []byte) error {
		req := &queuepb.Request{}
		require.NoError(t, proto.Unmarshal(data, req))

		require.Equal(t, string(messageData), req.GetBody())
		require.Equal(t, handlerData, req.GetHandler())

		// Compare headers
		require.Equal(t, len(req.GetHeaders()), len(headersData))
		for k, v := range headersData {
			header, ok := req.GetHeaders()[k]
			require.True(t, ok, "Header %s not found", k)
			require.ElementsMatch(t, v, header.GetValues())
		}

		return nil
	}

	// WithSendToKafka
	c, err := ammoclient.New(ammoclient.WithSendToKafka(checkFunc))
	require.NoError(t, err)

	require.NoError(t, c.SendHTTPRequest(
		ctx, []byte(messageData), handlerData, headersData))

	// WithSendToKafkaSarama
	testCh := make(chan *sarama.ProducerMessage, 1)

	c, err = ammoclient.New(ammoclient.WithSendToKafkaSarama(testCh, "test-topic"))
	require.NoError(t, err)

	require.NoError(t, c.SendHTTPRequest(
		ctx, []byte(messageData), handlerData, headersData))

	msg := <-testCh
	require.NotNil(t, msg)
	require.Equal(t, "test-topic", msg.Topic)
	data, err := msg.Value.Encode()
	require.NoError(t, err)
	require.NoError(t, checkFunc(ctx, data))

	// Test error cases
	require.Error(t, c.SendHTTPRequest(ctx, nil, handlerData, headersData))
	require.Error(t, c.SendHTTPRequest(ctx, []byte(messageData), "", headersData))
}

func TestPassRate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const messageData = "test message"
	headersData := map[string][]string{"Content-Type": {"application/json"}}

	// Test invalid pass rates
	_, err := ammoclient.New(
		ammoclient.WithSendToKafka(func(ctx context.Context, data []byte) error { return nil }),
		ammoclient.WithPassRate(-0.1),
	)
	require.Error(t, err)

	_, err = ammoclient.New(
		ammoclient.WithSendToKafka(func(ctx context.Context, data []byte) error { return nil }),
		ammoclient.WithPassRate(1.1),
	)
	require.Error(t, err)

	// Test passRate = 0 (all requests should be ignored)
	processed := 0
	c, err := ammoclient.New(
		ammoclient.WithSendToKafka(func(ctx context.Context, data []byte) error {
			processed++
			return nil
		}),
		ammoclient.WithPassRate(0),
	)
	require.NoError(t, err)

	for range 100 {
		err := c.SendHTTPRequest(ctx, []byte(messageData), "handler", headersData)
		require.NoError(t, err)
	}
	require.Equal(t, 0, processed, "with passRate=0, no requests should be processed")

	// Test passRate = 1 (all requests should be processed)
	processed = 0
	c, err = ammoclient.New(
		ammoclient.WithSendToKafka(func(ctx context.Context, data []byte) error {
			processed++
			return nil
		}),
		ammoclient.WithPassRate(1),
	)
	require.NoError(t, err)

	for range 100 {
		err := c.SendHTTPRequest(ctx, []byte(messageData), "handler", headersData)
		require.NoError(t, err)
	}
	require.Equal(t, 100, processed, "with passRate=1, all requests should be processed")

	// Test passRate = 0.5 (approximately half of requests should be processed)
	processed = 0
	c, err = ammoclient.New(
		ammoclient.WithSendToKafka(func(ctx context.Context, data []byte) error {
			processed++
			return nil
		}),
		ammoclient.WithPassRate(0.5),
	)
	require.NoError(t, err)

	for range 1000 {
		err := c.SendHTTPRequest(ctx, []byte(messageData), "handler", headersData)
		require.NoError(t, err)
	}

	// With 1000 requests and passRate=0.5, we expect roughly 500 requests to be processed
	// Allow for some statistical variance (±10%)
	expectedMin := 450
	expectedMax := 550
	require.Greater(t, processed, expectedMin, "processed count should be > %d", expectedMin)
	require.Less(t, processed, expectedMax, "processed count should be < %d", expectedMax)
}
