package pubsub

import (
	"context"
	"errors"
	"sync"

	"github.com/segmentio/kafka-go"
)

var Closed = errors.New("pubsub: closed")

type Handler func(ctx context.Context, msg Message) error

type Subscriber struct {
	reader *kafka.Reader
	errCh  chan error
	doneCh chan struct{}
	begin  sync.Once
	end    sync.Once
}

func NewSubscriber(r *kafka.Reader) *Subscriber {
	return &Subscriber{
		reader: r,
		errCh:  make(chan error, 1),
		doneCh: make(chan struct{}),
	}
}

// Receive handles the message received from the message queue.
// Returning an error will not commit the offset.
func (s *Subscriber) Receive(ctx context.Context, h Handler) func() error {
	s.begin.Do(func() {
		go func() {
			for {
				err := s.receive(ctx, h)
				if errors.Is(err, Closed) {
					s.errCh <- err
					close(s.errCh)
					return
				}
			}
		}()
	})

	return s.stop
}

func (s *Subscriber) receive(ctx context.Context, h Handler) error {
	for {
		select {
		case <-s.doneCh:
			return Closed
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

func (s *Subscriber) stop() (err error) {
	s.end.Do(func() {
		close(s.doneCh)
		err = <-s.errCh
	})

	return
}
