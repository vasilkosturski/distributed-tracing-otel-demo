package messaging

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// MessageProducer defines the interface for publishing messages to Kafka.
// This is an infrastructure concern that deals with external message brokers.
type MessageProducer interface {
	WriteMessage(ctx context.Context, msg kafka.Message) error
	Close() error
}

// MessageConsumer defines the interface for consuming messages from Kafka.
// This is an infrastructure concern that deals with external message brokers.
type MessageConsumer interface {
	ReadMessage(ctx context.Context) (*kafka.Message, error)
	Close() error
}
