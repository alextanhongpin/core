package pubsub

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, msg Message) error

type Subscriber struct {
	reader *kafka.Reader
}

func NewSubscriber(r *kafka.Reader) *Subscriber {
	return &Subscriber{
		reader: r,
	}
}

// Receive handles the message received from the message queue.
// Returning an error will not commit the offset.
func (s *Subscriber) Receive(ctx context.Context, h Handler) (func(), <-chan error) {
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	errCh := make(chan error)
	stop := func() {
		cancel()

		wg.Wait()
	}

	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(errCh)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			case errCh <- s.receive(ctx, h):
			}
		}
	}()

	return stop, errCh
}

func (s *Subscriber) receive(ctx context.Context, h Handler) error {
	for {
		msg, err := s.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		if err := h(ctx, NewMessage(msg)); err != nil {
			return err
		}

		if err := s.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}
