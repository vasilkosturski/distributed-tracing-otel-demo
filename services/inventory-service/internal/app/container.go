package app

import (
	"context"
	"fmt"
	"inventoryservice/internal/config"
	"inventoryservice/internal/platform/kafka"
	"inventoryservice/internal/platform/observability"
	"os"

	otelkafka "github.com/Trendyol/otel-kafka-konsumer"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Container holds expensive-to-create singleton resources and dependencies
type Container struct {
	config            *config.Config
	logger            observability.Logger
	tracer            observability.Tracer
	messageConsumer   kafka.Consumer
	messageProducer   kafka.Producer
	otelLogShutdown   func(context.Context) error
	otelTraceShutdown func(context.Context) error
}

// NewContainer creates and initializes all infrastructure components
func NewContainer(ctx context.Context) (*Container, error) {
	// Load configuration first
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	container := &Container{
		config: cfg,
	}

	// Initialize logger
	if err := container.setupLogger(ctx); err != nil {
		return nil, err
	}

	// Setup OpenTelemetry and Kafka
	if err := container.setupObservability(ctx); err != nil {
		return nil, err
	}

	return container, nil
}

// setupLogger initializes the logger with OpenTelemetry integration
func (c *Container) setupLogger(ctx context.Context) error {
	// Start with basic logger
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	c.logger = logger
	return nil
}

// setupObservability configures OpenTelemetry logging and tracing
func (c *Container) setupObservability(ctx context.Context) error {
	// Setup logging SDK
	otelLogShutdown, err := observability.SetupLoggingSDK(ctx, c.config)
	if err != nil {
		c.logger.Error("Failed to setup OpenTelemetry logging", zap.Error(err))
	}
	c.otelLogShutdown = otelLogShutdown

	// Setup tracing SDK
	tp, otelTraceShutdown, err := observability.SetupTracingSDK(ctx, c.config)
	if err != nil {
		c.logger.Error("Failed to setup OpenTelemetry tracing", zap.Error(err))
	}
	c.otelTraceShutdown = otelTraceShutdown

	// Re-initialize logger with OTel bridge
	c.reinitializeLoggerWithOTel()

	// Setup tracer
	c.tracer = otel.Tracer(config.ServiceName)

	// Setup Kafka with the TracerProvider
	if err := c.setupKafkaWithTracer(tp); err != nil {
		return err
	}

	return nil
}

// reinitializeLoggerWithOTel creates a new logger with OpenTelemetry integration
func (c *Container) reinitializeLoggerWithOTel() {
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
	logger := zap.New(finalCore,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.String("service.name", config.ServiceName)),
	)

	c.logger = logger
	c.logger.Info("Logger re-initialized with OpenTelemetry bridge")
}

// setupKafkaWithTracer initializes Kafka consumer and producer with OpenTelemetry
func (c *Container) setupKafkaWithTracer(tp trace.TracerProvider) error {
	// Create Kafka reader
	readerConfig := kafkago.ReaderConfig{
		Brokers: []string{c.config.KafkaBroker},
		Topic:   config.OrderCreatedTopic,
		GroupID: config.GroupID,
	}

	baseReader := kafkago.NewReader(readerConfig)
	reader, err := otelkafka.NewReader(baseReader)
	if err != nil {
		return err
	}
	c.messageConsumer = reader

	// Create Kafka writer
	baseWriter := &kafkago.Writer{
		Addr:         kafkago.TCP(c.config.KafkaBroker),
		Topic:        config.InventoryTopic,
		Balancer:     &kafkago.LeastBytes{},
		BatchTimeout: config.BatchTimeout,
		BatchSize:    config.BatchSize,
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
	c.messageProducer = writer

	return nil
}

// Shutdown gracefully shuts down all infrastructure components
func (c *Container) Shutdown(ctx context.Context) {
	c.logger.Info("Shutting down infrastructure...")

	// Close Kafka components
	if c.messageConsumer != nil {
		if err := c.messageConsumer.Close(); err != nil {
			c.logger.Error("Failed to close message consumer", zap.Error(err))
		}
	}

	if c.messageProducer != nil {
		if err := c.messageProducer.Close(); err != nil {
			c.logger.Error("Failed to close message producer", zap.Error(err))
		}
	}

	// Shutdown OpenTelemetry
	if c.otelTraceShutdown != nil {
		if err := c.otelTraceShutdown(ctx); err != nil {
			c.logger.Error("Failed to shutdown OTel tracing", zap.Error(err))
		}
	}

	if c.otelLogShutdown != nil {
		if err := c.otelLogShutdown(ctx); err != nil {
			c.logger.Error("Failed to shutdown OTel logging", zap.Error(err))
		}
	}

	// Sync logger
	if err := c.logger.Sync(); err != nil {
		// Can't log this error since logger might be closed
		fmt.Printf("Failed to sync logger: %v\n", err)
	}

	c.logger.Info("Infrastructure shutdown complete")
}

// Getters for accessing infrastructure components
func (c *Container) Logger() observability.Logger    { return c.logger }
func (c *Container) Tracer() observability.Tracer    { return c.tracer }
func (c *Container) MessageConsumer() kafka.Consumer { return c.messageConsumer }
func (c *Container) MessageProducer() kafka.Producer { return c.messageProducer }
