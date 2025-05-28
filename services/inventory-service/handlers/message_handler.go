package handlers

import (
	"context"
	"encoding/json"

	"inventoryservice/events"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

// Logger interface defines what the handler needs for logging
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// InventoryService interface defines the business logic the handler depends on
type InventoryService interface {
	ProcessOrderCreated(ctx context.Context, order events.OrderCreatedEvent) (*events.InventoryReservedEvent, error)
}

// MessageProducer interface defines message publishing capabilities
type MessageProducer interface {
	WriteMessage(ctx context.Context, msg kafka.Message) error
}

// MessageHandler handles Kafka message processing
type MessageHandler struct {
	inventoryService InventoryService
	producer         MessageProducer
	logger           Logger
}

// NewMessageHandler creates a new MessageHandler instance with explicit dependencies
func NewMessageHandler(inventoryService InventoryService, producer MessageProducer, logger Logger) *MessageHandler {
	return &MessageHandler{
		inventoryService: inventoryService,
		producer:         producer,
		logger:           logger,
	}
}

// HandleOrderCreated processes an OrderCreated message from Kafka
func (h *MessageHandler) HandleOrderCreated(ctx context.Context, msg kafka.Message) error {
	// Extract trace context from the incoming Kafka message headers
	// This is important for connecting the trace from the producer
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}

	// Extract headers from Kafka message to our carrier
	for _, header := range msg.Headers {
		carrier[string(header.Key)] = string(header.Value)
	}

	// Extract the context from the carrier - this will have the parent span info
	msgCtx := propagator.Extract(ctx, carrier)

	h.logger.Info("📨 Raw Kafka message received",
		zap.ByteString("key", msg.Key),
		zap.Int("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
	)

	// Deserialize the order event
	var order events.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		h.logger.Error("❌ Invalid JSON in OrderCreated event",
			zap.Error(err),
			zap.ByteString("raw_value", msg.Value),
		)
		return err
	}

	h.logger.Info("✅ Received OrderCreated event processed", zap.String("order_id", order.OrderID))

	// Process the order through the inventory service
	reservedEvent, err := h.inventoryService.ProcessOrderCreated(msgCtx, order)
	if err != nil {
		h.logger.Error("❌ Failed to process order", zap.Error(err), zap.String("order_id", order.OrderID))
		return err
	}

	h.logger.Info("✅ Inventory reserved", zap.String("order_id", reservedEvent.OrderID))

	// Serialize the response event
	payload, err := json.Marshal(*reservedEvent)
	if err != nil {
		h.logger.Error("❌ Failed to serialize InventoryReserved event",
			zap.Error(err),
			zap.String("order_id", order.OrderID),
		)
		return err
	}

	// Create a message with context that will propagate the trace
	kafkaMsg := kafka.Message{
		Value: payload,
		Key:   []byte(order.OrderID),
	}

	if err := h.producer.WriteMessage(msgCtx, kafkaMsg); err != nil {
		h.logger.Error("❌ Failed to publish InventoryReserved event",
			zap.Error(err),
			zap.String("order_id", order.OrderID),
		)
		return err
	}

	h.logger.Info("📤 Sent InventoryReserved event", zap.String("order_id", order.OrderID))
	return nil
}
