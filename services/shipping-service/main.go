package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

var (
	kafkaBrokerAddress = "kafka:9093"
	orderCreatedTopic  = "OrderCreated"
	packagingTopic     = "PackagingCompleted"
	groupID            = "shipping-service-group"
)

type OrderCreatedEvent struct {
	OrderID string `json:"order_id"`
}

type PackagingCompletedEvent struct {
	OrderID string `json:"order_id"`
}

func main() {
	// Create base zap logger
	baseLogger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("failed to create zap logger: %w", err))
	}
	defer baseLogger.Sync()

	// Wrap it with otelzap to enable trace correlation
	otelLogger := otelzap.New(baseLogger)
	defer otelLogger.Sync()

	// Create context + logger
	ctx := context.Background()
	log := otelLogger.Ctx(ctx)

	fmt.Println("Shipping Service is starting...")

	log.Info("Connecting to Kafka", zap.String("broker", kafkaBrokerAddress))
	log.Info("Configured topics",
		zap.String("consumer_topic", orderCreatedTopic),
		zap.String("producer_topic", packagingTopic),
	)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBrokerAddress},
		Topic:   orderCreatedTopic,
		GroupID: groupID,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			log.Error("Error closing Kafka reader", zap.Error(err))
		}
	}()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokerAddress),
		Topic:    packagingTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Error("Error closing Kafka writer", zap.Error(err))
		}
	}()

	log.Info("Kafka consumer started. Waiting for messages...")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Error("❌ Error reading from Kafka", zap.Error(err))
			continue
		}

		log.Info("📨 Raw Kafka message",
			zap.ByteString("key", msg.Key),
			zap.ByteString("value", msg.Value),
		)

		var order OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			log.Error("❌ Invalid JSON in OrderCreated event", zap.Error(err))
			continue
		}

		log.Info("✅ Received OrderCreated", zap.String("order_id", order.OrderID))

		out := PackagingCompletedEvent{OrderID: order.OrderID}
		payload, err := json.Marshal(out)
		if err != nil {
			log.Error("❌ Failed to serialize PackagingCompleted event", zap.Error(err))
			continue
		}

		err = writer.WriteMessages(ctx, kafka.Message{Value: payload})
		if err != nil {
			log.Error("❌ Failed to publish PackagingCompleted", zap.Error(err))
		} else {
			log.Info("📤 Sent PackagingCompleted", zap.String("order_id", order.OrderID))
		}
	}
}
