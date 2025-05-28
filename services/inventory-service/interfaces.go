package main

import (
	"context"
	"inventoryservice/events"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logger interface defines what our services need for logging
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	With(fields ...zap.Field) *zap.Logger
	Sync() error
}

// InventoryService interface defines inventory business logic
type InventoryService interface {
	ProcessOrderCreated(ctx context.Context, order events.OrderCreatedEvent) (*events.InventoryReservedEvent, error)
}

// MessageProducer interface defines message publishing capabilities
type MessageProducer interface {
	WriteMessage(ctx context.Context, msg kafka.Message) error
	Close() error
}

// MessageConsumer interface defines message consuming capabilities
type MessageConsumer interface {
	ReadMessage(ctx context.Context) (*kafka.Message, error)
	Close() error
}

// MessageHandler interface defines message processing
type MessageHandler interface {
	HandleOrderCreated(ctx context.Context, msg kafka.Message) error
}

// Tracer interface wraps OpenTelemetry tracer
type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}
