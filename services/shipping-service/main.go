package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
	ctx := context.Background()

	// 1. Build Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("shipping-service"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build resource: %v\n", err)
		os.Exit(1)
	}

	// 2. OTLP Log Exporter to Grafana Cloud
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint("otlp-gateway-prod-eu-west-2.grafana.net:4317"),
		otlploggrpc.WithHeaders(map[string]string{
			"Authorization": "Basic MTE5NzE2NzpnbGNfZXlK...", // truncate in real code
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create OTLP exporter: %v\n", err)
		os.Exit(1)
	}

	// 3. LoggerProvider with batch processor and resource
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	defer loggerProvider.Shutdown(ctx)

	// 4. Register as global logger provider
	global.SetLoggerProvider(loggerProvider)

	// 5. Create zap logger and bridge it with otelzap
	baseLogger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("failed to create zap logger: %w", err))
	}
	defer baseLogger.Sync()

	otelLogger := otelzap.New(baseLogger)
	defer otelLogger.Sync()
	log := otelLogger.Ctx(ctx)

	// --- Service Logs ---
	log.Info("Shipping Service is starting...")
	log.Info("Connecting to Kafka", zap.String("broker", kafkaBrokerAddress))
	log.Info("Configured topics",
		zap.String("consumer_topic", orderCreatedTopic),
		zap.String("producer_topic", packagingTopic),
	)

	// --- Kafka Setup ---
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBrokerAddress},
		Topic:   orderCreatedTopic,
		GroupID: groupID,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokerAddress),
		Topic:    packagingTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

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
