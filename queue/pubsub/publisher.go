package pubsub

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Publisher struct {
	writer *kafka.Writer
	topic  string
}

func NewPublisher(w *kafka.Writer) *Publisher {
	return &Publisher{
		writer: w,
	}
}

func (p *Publisher) Publish(ctx context.Context, msgs ...Message) error {
	m := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		m[i] = kafka.Message{
			Key:   msg.Key(),
			Value: msg.Value(),
		}
	}

	return p.writer.WriteMessages(ctx, m...)
}
