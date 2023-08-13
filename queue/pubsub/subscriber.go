package pubsub

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, msg Message) error

type Subscriber struct {
	reader *kafka.Reader
	errCh  chan error
	doneCh chan struct{}
	begin  sync.Once
	end    sync.Once
	wg     sync.WaitGroup
}

func NewSubscriber(r *kafka.Reader) *Subscriber {
	return &Subscriber{
		reader: r,
		errCh:  make(chan error),
		doneCh: make(chan struct{}),
	}
}

// Receive handles the message received from the message queue.
// Returning an error will not commit the offset.
func (s *Subscriber) Receive(ctx context.Context, h Handler) (func(), <-chan error) {
	s.begin.Do(func() {
		s.wg.Add(1)

		go func() {
			defer s.wg.Done()
			defer close(s.errCh)

			for {
				select {
				case <-s.doneCh:
					return
				case s.errCh <- s.receive(ctx, h):
				}
			}
		}()
	})

	return s.stop, s.errCh
}

func (s *Subscriber) receive(ctx context.Context, h Handler) error {
	for {
		select {
		case <-s.doneCh:
			return nil
		default:
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
}

func (s *Subscriber) stop() {
	s.end.Do(func() {
		close(s.doneCh)
		s.wg.Wait()
	})

	return
}
