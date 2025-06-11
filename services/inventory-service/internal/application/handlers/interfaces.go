package handlers

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// MessageHandler defines the interface for processing incoming messages.
// This belongs to the application layer and orchestrates between domain and infrastructure.
type MessageHandler interface {
	HandleOrderCreated(ctx context.Context, msg kafka.Message) error
}
