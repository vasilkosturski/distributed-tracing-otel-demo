package inventory

import (
	"context"
	"encoding/json"

	"inventoryservice/internal/platform/kafka"
	"inventoryservice/internal/platform/observability"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

// MessageHandler defines the interface for processing incoming messages.
type MessageHandler interface {
	HandleOrderCreated(ctx context.Context, msg kafkago.Message) error
}

// KafkaMessageHandler handles Kafka message processing for inventory events
type KafkaMessageHandler struct {
	inventoryService *InventoryService
	producer         kafka.Producer
	logger           observability.Logger
}

// NewMessageHandler creates a new MessageHandler instance with explicit dependencies
func NewMessageHandler(inventoryService *InventoryService, producer kafka.Producer, logger observability.Logger) MessageHandler {
	return &KafkaMessageHandler{
		inventoryService: inventoryService,
		producer:         producer,
		logger:           logger,
	}
}

// HandleOrderCreated processes an OrderCreated message from Kafka
func (h *KafkaMessageHandler) HandleOrderCreated(ctx context.Context, msg kafkago.Message) error {
	// Extract trace context to connect spans across services
	msgCtx := h.extractTraceContext(ctx, msg.Headers)

	h.logger.Info("📨 Raw Kafka message received",
		zap.ByteString("key", msg.Key),
		zap.Int("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
	)

	// Deserialize the order event
	var order OrderCreatedEvent
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

	// Publish the inventory reserved event
	return h.publishInventoryReserved(msgCtx, reservedEvent)
}

// extractTraceContext extracts OpenTelemetry trace context from Kafka message headers
func (h *KafkaMessageHandler) extractTraceContext(ctx context.Context, headers []kafkago.Header) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}

	// Extract headers from Kafka message to our carrier
	for _, header := range headers {
		carrier[string(header.Key)] = string(header.Value)
	}

	// Extract the context from the carrier - this will have the parent span info
	return propagator.Extract(ctx, carrier)
}

// publishInventoryReserved publishes an InventoryReserved event to Kafka
func (h *KafkaMessageHandler) publishInventoryReserved(ctx context.Context, event *InventoryReservedEvent) error {
	// Serialize the response event
	payload, err := json.Marshal(*event)
	if err != nil {
		h.logger.Error("❌ Failed to serialize InventoryReserved event",
			zap.Error(err),
			zap.String("order_id", event.OrderID),
		)
		return err
	}

	// Create a message with context that will propagate the trace
	kafkaMsg := kafkago.Message{
		Value: payload,
		Key:   []byte(event.OrderID),
	}

	if err := h.producer.WriteMessage(ctx, kafkaMsg); err != nil {
		h.logger.Error("❌ Failed to publish InventoryReserved event",
			zap.Error(err),
			zap.String("order_id", event.OrderID),
		)
		return err
	}

	h.logger.Info("📤 Sent InventoryReserved event", zap.String("order_id", event.OrderID))
	return nil
}
