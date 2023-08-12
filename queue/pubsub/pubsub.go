package pubsub

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type publisher interface {
	Publish(ctx context.Context, msg Message) error
}

type subscriber interface {
	Subscribe(Handler)
}

type Message interface {
	Key() []byte
	Value() []byte
}

type KafkaMessage struct {
	kafka.Message
}

func NewMessage(msg kafka.Message) *KafkaMessage {
	return &KafkaMessage{
		Message: msg,
	}
}

func (k KafkaMessage) Key() []byte {
	return k.Message.Key
}

func (k KafkaMessage) Value() []byte {
	return k.Message.Value
}
