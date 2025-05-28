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
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Application holds all the components and manages the application lifecycle
type Application struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger

	// OpenTelemetry shutdown functions
	otelLogShutdown   func(context.Context) error
	otelTraceShutdown func(context.Context) error

	// Business components
	inventoryService *services.InventoryService
	messageHandler   *handlers.MessageHandler

	// Kafka components
	reader *otelkafka.Reader
	writer *otelkafka.Writer
}

// NewApplication creates a new Application instance
func NewApplication() *Application {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	return &Application{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize sets up all the application components
func (app *Application) Initialize() error {
	var err error

	// Initialize fallback logger
	app.logger, err = zap.NewProduction()
	if err != nil {
		return err
	}
	app.logger.Info("Inventory Service attempting to start (initial console logger)...")

	// Setup logging SDK
	app.otelLogShutdown, err = observability.SetupLoggingSDK(app.ctx)
	if err != nil {
		app.logger.Error("⚠️ Failed to setup OpenTelemetry SDK for logging completely", zap.Error(err))
	} else {
		app.logger.Info("✅ OpenTelemetry SDK for logging initialized (or partially).")
	}

	// Setup tracing SDK
	tp, otelTraceShutdown, err := observability.SetupTracingSDK(app.ctx)
	app.otelTraceShutdown = otelTraceShutdown
	if err != nil {
		app.logger.Error("⚠️ Failed to setup OpenTelemetry SDK for tracing completely", zap.Error(err))
	} else {
		app.logger.Info("✅ OpenTelemetry SDK for tracing initialized (or partially).")
	}

	// Re-initialize logger with OTel bridge
	logProvider := global.GetLoggerProvider()
	instrumentationScopeName := "inventory-service.manual"
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
	app.logger = zap.New(finalCore,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.String("service.name", config.ServiceName)),
	)
	app.logger.Info("🚀 Inventory Service Zap logger re-initialized with OTel bridge and console output.")

	// Initialize inventory service
	app.inventoryService = services.NewInventoryService(app.logger, config.ServiceName)

	// Setup Kafka components
	if err := app.setupKafka(tp); err != nil {
		return err
	}

	// Initialize message handler
	app.messageHandler = handlers.NewMessageHandler(app.inventoryService, app.writer, app.logger)

	return nil
}

// setupKafka initializes Kafka reader and writer with OTel instrumentation
func (app *Application) setupKafka(tp trace.TracerProvider) error {
	app.logger.Info("Connecting to Kafka", zap.String("broker", config.KafkaBrokerAddress))
	app.logger.Info("Configured topics",
		zap.String("consumer_topic", config.OrderCreatedTopic),
		zap.String("producer_topic", config.InventoryTopic),
	)

	// Create Kafka reader
	readerConfig := kafka.ReaderConfig{
		Brokers: []string{config.KafkaBrokerAddress},
		Topic:   config.OrderCreatedTopic,
		GroupID: config.GroupID,
	}

	baseReader := kafka.NewReader(readerConfig)
	reader, err := otelkafka.NewReader(baseReader)
	if err != nil {
		return err
	}
	app.reader = reader

	// Create Kafka writer
	baseWriter := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokerAddress),
		Topic:        config.InventoryTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

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
		return err
	}
	app.writer = writer

	return nil
}

// Run starts the main event processing loop
func (app *Application) Run() error {
	app.logger.Info("Kafka consumer started. Waiting for messages...")

	for {
		msg, err := app.reader.ReadMessage(app.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				app.logger.Info("Context done, exiting Kafka read loop.", zap.Error(err))
				break
			}
			app.logger.Error("❌ Error reading from Kafka", zap.Error(err))
			continue
		}

		// Handle the message using the message handler
		if err := app.messageHandler.HandleOrderCreated(app.ctx, *msg); err != nil {
			// Error is already logged in the handler, just continue to next message
			continue
		}
	}

	app.logger.Info("Inventory Service event loop finished. Shutting down...")
	return nil
}

// Shutdown gracefully shuts down all application components
func (app *Application) Shutdown() {
	app.logger.Info("Starting application shutdown...")

	// Cancel context
	if app.cancel != nil {
		app.cancel()
	}

	// Close Kafka components
	if app.reader != nil {
		app.logger.Info("Closing Kafka reader...")
		if err := app.reader.Close(); err != nil {
			app.logger.Error("❌ Failed to close Kafka reader", zap.Error(err))
		}
	}

	if app.writer != nil {
		app.logger.Info("Closing Kafka writer...")
		if err := app.writer.Close(); err != nil {
			app.logger.Error("❌ Failed to close Kafka writer", zap.Error(err))
		}
	}

	// Shutdown OpenTelemetry components
	if app.otelTraceShutdown != nil {
		app.logger.Info("Shutting down OpenTelemetry SDK for tracing...")
		if err := app.otelTraceShutdown(context.Background()); err != nil {
			app.logger.Error("❌ Error during OTel tracing SDK shutdown", zap.Error(err))
		}
		app.logger.Info("OTel tracing SDK shutdown complete.")
	}

	if app.otelLogShutdown != nil {
		app.logger.Info("Shutting down OpenTelemetry SDK for logging...")
		if err := app.otelLogShutdown(context.Background()); err != nil {
			app.logger.Error("❌ Error during OTel logging SDK shutdown", zap.Error(err))
		}
		app.logger.Info("OTel logging SDK shutdown complete.")
	}

	// Sync logger
	if app.logger != nil {
		_ = app.logger.Sync()
	}

	app.logger.Info("Application shutdown complete.")
}

func main() {
	app := NewApplication()
	defer app.Shutdown()

	if err := app.Initialize(); err != nil {
		stdlog.Fatalf("❌ Failed to initialize application: %v", err)
	}

	if err := app.Run(); err != nil {
		app.logger.Error("❌ Application error", zap.Error(err))
	}
}
