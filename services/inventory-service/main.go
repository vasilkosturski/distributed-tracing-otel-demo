package main

import (
	"context"
	"errors"
	"inventoryservice/config"
	"inventoryservice/handlers"
	"inventoryservice/observability"
	"inventoryservice/services"
	stdlog "log" // Alias standard log to prevent conflict
	"os"
	"os/signal"
	"time"

	// Local config package

	otelkafka "github.com/Trendyol/otel-kafka-konsumer" // OpenTelemetry Kafka instrumentation
	"github.com/segmentio/kafka-go"

	"go.opentelemetry.io/contrib/bridges/otelzap" // Official OTel Contrib bridge for Zap
	"go.opentelemetry.io/otel/attribute"

	// OTLP HTTP for Traces
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"             // SDK for traces
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0" // Add this for SpanFromContext
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	AppLogger *zap.Logger // Global ZapLogger
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	var err error

	AppLogger, err = zap.NewProduction() // Fallback console logger
	if err != nil {
		stdlog.Fatalf("❌ Failed to initialize fallback zap logger: %v\n", err)
	}
	defer func() {
		if AppLogger != nil {
			_ = AppLogger.Sync()
		}
	}()
	AppLogger.Info("Inventory Service attempting to start (initial console logger)...")

	// Setup logging SDK
	otelLogShutdown, otelLogSetupErr := observability.SetupLoggingSDK(ctx)
	if otelLogSetupErr != nil {
		AppLogger.Error("⚠️ Failed to setup OpenTelemetry SDK for logging completely", zap.Error(otelLogSetupErr))
	} else {
		AppLogger.Info("✅ OpenTelemetry SDK for logging initialized (or partially).")
	}
	if otelLogShutdown != nil {
		defer func() {
			AppLogger.Info("Shutting down OpenTelemetry SDK for logging...")
			if shutdownErr := otelLogShutdown(context.Background()); shutdownErr != nil {
				AppLogger.Error("❌ Error during OTel logging SDK shutdown", zap.Error(shutdownErr))
			}
			AppLogger.Info("OTel logging SDK shutdown complete.")
		}()
	}

	// Setup tracing SDK
	tp, otelTraceShutdown, otelTraceSetupErr := observability.SetupTracingSDK(ctx)
	if otelTraceSetupErr != nil {
		AppLogger.Error("⚠️ Failed to setup OpenTelemetry SDK for tracing completely", zap.Error(otelTraceSetupErr))
	} else {
		AppLogger.Info("✅ OpenTelemetry SDK for tracing initialized (or partially).")
	}
	if otelTraceShutdown != nil {
		defer func() {
			AppLogger.Info("Shutting down OpenTelemetry SDK for tracing...")
			if shutdownErr := otelTraceShutdown(context.Background()); shutdownErr != nil {
				AppLogger.Error("❌ Error during OTel tracing SDK shutdown", zap.Error(shutdownErr))
			}
			AppLogger.Info("OTel tracing SDK shutdown complete.")
		}()
	}

	// Re-initialize AppLogger with OTel bridge (official contrib) and console tee
	logProvider := global.GetLoggerProvider()              // Will be a no-op if OTel setup failed
	instrumentationScopeName := "inventory-service.manual" // Customize this
	otelZapCore := otelzap.NewCore(instrumentationScopeName,
		otelzap.WithLoggerProvider(logProvider),
	)

	consoleEncoderConfig := zap.NewProductionEncoderConfig()
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(consoleEncoderConfig),
		zapcore.Lock(os.Stdout),
		zap.InfoLevel,
	)

	finalCore := zapcore.NewTee(otelZapCore, consoleCore)
	AppLogger = zap.New(finalCore,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.String("service.name", config.ServiceName)),
	)
	AppLogger.Info("🚀 Inventory Service Zap logger re-initialized with OTel bridge and console output.")

	// Initialize inventory service
	inventoryService := services.NewInventoryService(AppLogger, config.ServiceName)

	// --- Service Logs ---
	AppLogger.Info("Connecting to Kafka", zap.String("broker", config.KafkaBrokerAddress))
	AppLogger.Info("Configured topics",
		zap.String("consumer_topic", config.OrderCreatedTopic),
		zap.String("producer_topic", config.InventoryTopic),
	)

	// --- Kafka Setup with OTel instrumentation ---
	// Create a base reader config
	readerConfig := kafka.ReaderConfig{
		Brokers: []string{config.KafkaBrokerAddress},
		Topic:   config.OrderCreatedTopic,
		GroupID: config.GroupID,
	}

	// Create the base reader first, then wrap with OTel instrumentation
	baseReader := kafka.NewReader(readerConfig)
	reader, err := otelkafka.NewReader(baseReader)
	if err != nil {
		AppLogger.Error("❌ Failed to create instrumented Kafka reader", zap.Error(err))
		return
	}

	defer func() {
		AppLogger.Info("Closing Kafka reader...")
		if err := reader.Close(); err != nil {
			AppLogger.Error("❌ Failed to close Kafka reader", zap.Error(err))
		}
	}()

	baseWriter := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokerAddress),
		Topic:        config.InventoryTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond, // Quick batching
		BatchSize:    100,                   // Small batch size for low latency
	}

	// Then create the writer with the direct tp instance
	writer, err := otelkafka.NewWriter(baseWriter,
		otelkafka.WithTracerProvider(tp),
		otelkafka.WithPropagator(propagation.TraceContext{}),
		otelkafka.WithAttributes(
			[]attribute.KeyValue{
				semconv.MessagingDestinationNameKey.String(config.InventoryTopic),
				attribute.String("messaging.kafka.client_id", config.ServiceName),
			},
		),
	)
	if err != nil {
		AppLogger.Error("❌ Failed to create instrumented Kafka writer", zap.Error(err))
		return
	}

	defer func() {
		AppLogger.Info("Closing Kafka writer...")
		if err := writer.Close(); err != nil {
			AppLogger.Error("❌ Failed to close Kafka writer", zap.Error(err))
		}
	}()

	// Initialize message handler
	messageHandler := handlers.NewMessageHandler(inventoryService, writer, AppLogger)

	AppLogger.Info("Kafka consumer started. Waiting for messages...")

	for {
		msg, err := reader.ReadMessage(ctx) // Using the cancellable context
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				AppLogger.Info("Context done, exiting Kafka read loop.", zap.Error(err))
				break
			}
			AppLogger.Error("❌ Error reading from Kafka", zap.Error(err))
			continue
		}

		// Handle the message using the message handler
		if err := messageHandler.HandleOrderCreated(ctx, *msg); err != nil {
			// Error is already logged in the handler, just continue to next message
			continue
		}
	}

	AppLogger.Info("Inventory Service event loop finished. Shutting down...")
}
