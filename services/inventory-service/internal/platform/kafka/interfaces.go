package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	WriteMessage(ctx context.Context, msg kafka.Message) error
	Close() error
}

type Consumer interface {
	ReadMessage(ctx context.Context) (*kafka.Message, error)
	Close() error
}
