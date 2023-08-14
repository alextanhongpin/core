package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/alextanhongpin/core/queue/pubsub"
	"github.com/segmentio/kafka-go"
)

const kafkaHost = "localhost:9093"

const (
	eventsTopic      = "events"
	eventsRetryTopic = "events-retry"
	consumerGroup    = "consumers/events"
)

var EventPublisher = pubsub.NewPublisher(
	// NOTE: kafka.NewWriter is deprecated.
	&kafka.Writer{
		Addr:     kafka.TCP(kafkaHost),
		Topic:    eventsTopic,
		Balancer: &kafka.Hash{},
	},
)

var EventRetryPublisher = pubsub.NewPublisher(
	&kafka.Writer{
		Addr:     kafka.TCP(kafkaHost),
		Topic:    eventsRetryTopic,
		Balancer: &kafka.Hash{},
	},
)

var EventSubscriber = pubsub.NewSubscriber(
	kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaHost},
		GroupID: consumerGroup,
		Topic:   eventsTopic,
	}),
)

var EventRetrySubscriber = pubsub.NewSubscriber(
	kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaHost},
		GroupID: consumerGroup,
		Topic:   eventsRetryTopic,
	}),
)

func main() {
	var isConsumer bool
	flag.BoolVar(&isConsumer, "consumer", false, "Whether this is consumer or not")
	flag.Parse()

	ctx := context.Background()

	if isConsumer {
		fmt.Println("subscribing ...")

		ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		stop, errCh := EventSubscriber.Receive(ctx, func(ctx context.Context, msg pubsub.Message) error {
			fmt.Printf("received: key=%s value=%s\n", msg.Key(), msg.Value())
			return EventRetryPublisher.Publish(ctx, msg)
		})
		defer stop()

		stopRetry, retryErrCh := EventRetrySubscriber.Receive(ctx, func(ctx context.Context, msg pubsub.Message) error {
			kmsg := pubsub.AsKafkaMessage(msg)
			// Retry after 10 seconds.
			sleep := kmsg.Time.Add(10 * time.Second).Sub(time.Now())
			fmt.Printf("received dead letter: key=%s value=%s retry-after=%s\n", msg.Key(), msg.Value(), sleep)
			time.Sleep(sleep)

			fmt.Println("retried successfully")
			// TODO: Save to database if still fails.
			return nil
		})
		defer stopRetry()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("terminated")
				return
			case err := <-errCh:
				log.Println(err)
			case err := <-retryErrCh:
				log.Println(err)
			}
		}
	} else {
		fmt.Println("publishing ...")

		err := EventPublisher.Publish(ctx, pubsub.NewMessage(kafka.Message{
			Key:   []byte("hello"),
			Value: []byte("world"),
		}))
		if err != nil {
			panic(err)
		}

		fmt.Println("published")
	}
}
