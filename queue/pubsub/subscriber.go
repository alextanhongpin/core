package pubsub

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, msg Message) error

type Subscriber struct {
	reader  *kafka.Reader
	handler Handler
	errCh   chan error
	begin   sync.Once
	end     sync.Once
}

type SubscriberOption struct {
	Reader  *kafka.Reader
	Handler Handler
}

func NewSubscriber(opt SubscriberOption) *Subscriber {
	return &Subscriber{
		reader:  opt.Reader,
		errCh:   make(chan error, 1),
		handler: opt.Handler,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context) func() error {
	stop := func() error {
		return nil
	}

	s.begin.Do(func() {
		ctx, cancel := context.WithCancel(ctx)

		go func() {
			s.errCh <- s.subscribe(ctx)
			close(s.errCh)
		}()

		stop = func() error {
			cancel()

			return s.stop()
		}
	})

	return stop
}

func (s *Subscriber) subscribe(ctx context.Context) error {
	for {
		msg, err := s.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		if err := s.handler(ctx, NewMessage(msg)); err != nil {
			return err
		}

		if err := s.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

func (s *Subscriber) stop() (err error) {
	s.end.Do(func() {
		err = <-s.errCh
	})

	return
}
