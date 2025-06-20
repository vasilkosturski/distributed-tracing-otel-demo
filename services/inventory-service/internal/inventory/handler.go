package inventory

import (
	"context"
	"encoding/json"

	"inventoryservice/internal/platform/kafka"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

type MessageHandler interface {
	HandleOrderCreated(ctx context.Context, msg kafkago.Message) error
}

type KafkaMessageHandler struct {
	inventoryService *InventoryService
	producer         kafka.Producer
	logger           *zap.Logger
}

func NewMessageHandler(inventoryService *InventoryService, producer kafka.Producer, logger *zap.Logger) MessageHandler {
	return &KafkaMessageHandler{
		inventoryService: inventoryService,
		producer:         producer,
		logger:           logger,
	}
}

func (h *KafkaMessageHandler) HandleOrderCreated(ctx context.Context, msg kafkago.Message) error {
	msgCtx := h.extractTraceContext(ctx, msg.Headers)

	h.logger.Info("📨 Raw Kafka message received",
		zap.ByteString("key", msg.Key),
		zap.Int("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
	)

	var order OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		h.logger.Error("❌ Invalid JSON in OrderCreated event",
			zap.Error(err),
			zap.ByteString("raw_value", msg.Value),
		)
		return err
	}

	h.logger.Info("✅ Received OrderCreated event processed", zap.String("order_id", order.OrderID))

	reservedEvent, err := h.inventoryService.ProcessOrderCreated(msgCtx, order)
	if err != nil {
		h.logger.Error("❌ Failed to process order", zap.Error(err), zap.String("order_id", order.OrderID))
		return err
	}

	h.logger.Info("✅ Inventory reserved", zap.String("order_id", reservedEvent.OrderID))

	return h.publishInventoryReserved(msgCtx, reservedEvent)
}

func (h *KafkaMessageHandler) extractTraceContext(ctx context.Context, headers []kafkago.Header) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}

	for _, header := range headers {
		carrier[string(header.Key)] = string(header.Value)
	}

	return propagator.Extract(ctx, carrier)
}

func (h *KafkaMessageHandler) publishInventoryReserved(ctx context.Context, event *InventoryReservedEvent) error {
	payload, err := json.Marshal(*event)
	if err != nil {
		h.logger.Error("❌ Failed to serialize InventoryReserved event",
			zap.Error(err),
			zap.String("order_id", event.OrderID),
		)
		return err
	}

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
