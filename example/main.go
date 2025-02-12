//nolint:govet,gosec,lostcancel //ok
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/n-r-w/ammo-collector/example/api"
	"github.com/n-r-w/ammo-collector/internal/config"
	"github.com/n-r-w/ammo-collector/internal/pb/api/collector"
	"github.com/n-r-w/ammo-collector/pkg/ammoclient"
	"github.com/n-r-w/grpcsrv/grpcdial"
	"github.com/n-r-w/kafkaclient/producer"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
)

func main() {
	ctx := context.Background()

	// Parse command line flags
	intervalRequestFlag := flag.Duration("req", time.Second, "Interval between requests (e.g. 1s, 500ms)")
	intervalTaskFlag := flag.Duration("task", time.Second*10, "Interval between tasks (e.g. 1s, 500ms)")
	limitFlag := flag.Int("limit", 1000, "Limit of requests per collection")
	flag.Parse()

	// Register shutdown signal
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Load config from environment variables
	cfg := config.MustNew()

	// Run async producer
	asyncProducer, err := runAsyncProducer(ctx, cfg.Kafka.KafkaBrokers)
	if err != nil {
		log.Printf("Failed to start async producer: %v", err)
		return
	}

	var wg sync.WaitGroup

	// Start sending messages
	sendMessagesCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	err = sendMessages(sendMessagesCtx, &wg, *intervalRequestFlag, cfg.Kafka.KafkaTopic, asyncProducer.Input())
	if err != nil {
		log.Printf("Failed to send messages: %v", err)
		return
	}

	// Start sending tasks
	if err := sendTasks(sendMessagesCtx, &wg, cfg, *intervalTaskFlag, *limitFlag); err != nil {
		log.Printf("Failed to send tasks: %v", err)
		return
	}

	// Wait for shutdown signal
	<-sigchan
	log.Println("Shutting down...")

	// Stop send messages
	cancel()
	wg.Wait()

	// Stop async producer
	if err := asyncProducer.Stop(ctx); err != nil {
		log.Printf("Error stopping async producer: %v", err)
	}
	log.Println("Async producer stopped")
}

func runAsyncProducer(ctx context.Context, brokers []string) (*producer.AsyncProducer, error) {
	// Create and start producer
	p, err := producer.NewAsyncProducer(ctx, "example-service", brokers)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	if err := p.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start producer: %w", err)
	}
	log.Println("Async producer started")

	return p, nil
}

func sendMessages(
	ctx context.Context, wg *sync.WaitGroup, interval time.Duration, topic string, ch chan<- *sarama.ProducerMessage,
) error {
	wg.Add(1)

	c, err := ammoclient.New(ammoclient.WithSendToKafkaSarama(ch, topic))
	if err != nil {
		return fmt.Errorf("failed to create ammoclient: %w", err)
	}

	go func() {
		defer wg.Done()

		var i int
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			i++
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var randomBigMessage strings.Builder
				for range 100 {
					_, _ = randomBigMessage.WriteString(uuid.New().String())
				}

				if err := c.SendGRPCRequest(ctx,
					&api.TestMessage{
						Message: randomBigMessage.String(),
						Data:    int64(i),
					}, "handler", nil); err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}

					log.Printf("failed to send message: %v", err)
					return
				}

				log.Printf("sent message: %d", i)
			}
		}
	}()

	return nil
}

// sendTasks sends tasks to the collector.
func sendTasks(
	ctx context.Context, wg *sync.WaitGroup, cfg *config.Config, interval time.Duration, reqLimit int,
) error {
	wg.Add(1)
	dialer := grpcdial.New(ctx)

	conn, err := dialer.Dial(ctx,
		fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.GRPC.Port), "example",
		grpcdial.WithCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := collector.NewCollectionServiceClient(conn)

	go func() {
		defer wg.Done()
		defer func() {
			_ = dialer.Stop(context.WithoutCancel(ctx))
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				const (
					handlerName = "handler"
					timeLimit   = time.Minute
				)

				ctxCall, _ := context.WithTimeout(ctx, time.Second) //nolint:lostcancel //ok
				resp, err := client.CreateTask(ctxCall, &collector.CreateTaskRequest{
					SelectionCriteria: &collector.MessageSelectionCriteria{
						Handler: handlerName,
					},
					CompletionCriteria: &collector.CompletionCriteria{
						TimeLimit:         durationpb.New(timeLimit),
						RequestCountLimit: uint32(reqLimit),
					},
				})
				if err != nil {
					log.Printf("failed to create task: %v", err)
				}
				log.Printf("created task. collection id: %d", resp.GetCollectionId())
			}
		}
	}()

	return nil
}
