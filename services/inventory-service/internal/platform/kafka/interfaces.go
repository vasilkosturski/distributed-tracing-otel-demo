package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// Producer defines the interface for publishing messages to Kafka.
// This is a platform concern that deals with external message brokers.
type Producer interface {
	WriteMessage(ctx context.Context, msg kafka.Message) error
	Close() error
}

// Consumer defines the interface for consuming messages from Kafka.
// This is a platform concern that deals with external message brokers.
type Consumer interface {
	ReadMessage(ctx context.Context) (*kafka.Message, error)
	Close() error
}
