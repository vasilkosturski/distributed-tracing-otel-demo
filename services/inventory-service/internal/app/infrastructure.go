package app

import (
	"context"
	"fmt"
	"inventoryservice/internal/infrastructure/config"
	"inventoryservice/internal/infrastructure/messaging"
	infra_obs "inventoryservice/internal/infrastructure/observability"
	"os"

	otelkafka "github.com/Trendyol/otel-kafka-konsumer"
	"github.com/segmentio/kafka-go"
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

// Infrastructure holds expensive-to-create singleton resources
type Infrastructure struct {
	config            *config.Config
	logger            infra_obs.Logger
	tracer            infra_obs.Tracer
	messageConsumer   messaging.MessageConsumer
	messageProducer   messaging.MessageProducer
	otelLogShutdown   func(context.Context) error
	otelTraceShutdown func(context.Context) error
}

// NewInfrastructure creates and initializes all infrastructure components
func NewInfrastructure(ctx context.Context) (*Infrastructure, error) {
	// Load configuration first
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	infra := &Infrastructure{
		config: cfg,
	}

	// Initialize logger
	if err := infra.setupLogger(ctx); err != nil {
		return nil, err
	}

	// Setup OpenTelemetry and Kafka
	if err := infra.setupObservability(ctx); err != nil {
		return nil, err
	}

	return infra, nil
}

// setupLogger initializes the logger with OpenTelemetry integration
func (infra *Infrastructure) setupLogger(ctx context.Context) error {
	// Start with basic logger
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	infra.logger = logger
	return nil
}

// setupObservability configures OpenTelemetry logging and tracing
func (infra *Infrastructure) setupObservability(ctx context.Context) error {
	// Setup logging SDK
	otelLogShutdown, err := infra_obs.SetupLoggingSDK(ctx, infra.config)
	if err != nil {
		infra.logger.Error("Failed to setup OpenTelemetry logging", zap.Error(err))
	}
	infra.otelLogShutdown = otelLogShutdown

	// Setup tracing SDK
	tp, otelTraceShutdown, err := infra_obs.SetupTracingSDK(ctx, infra.config)
	if err != nil {
		infra.logger.Error("Failed to setup OpenTelemetry tracing", zap.Error(err))
	}
	infra.otelTraceShutdown = otelTraceShutdown

	// Re-initialize logger with OTel bridge
	infra.reinitializeLoggerWithOTel()

	// Setup tracer
	infra.tracer = otel.Tracer(config.ServiceName)

	// Setup Kafka with the TracerProvider
	if err := infra.setupKafkaWithTracer(tp); err != nil {
		return err
	}

	return nil
}

// reinitializeLoggerWithOTel creates a new logger with OpenTelemetry integration
func (infra *Infrastructure) reinitializeLoggerWithOTel() {
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

	infra.logger = logger
	infra.logger.Info("Logger re-initialized with OpenTelemetry bridge")
}

// setupKafkaWithTracer initializes Kafka consumer and producer with OpenTelemetry
func (infra *Infrastructure) setupKafkaWithTracer(tp trace.TracerProvider) error {
	// Create Kafka reader
	readerConfig := kafka.ReaderConfig{
		Brokers: []string{infra.config.KafkaBroker},
		Topic:   config.OrderCreatedTopic,
		GroupID: config.GroupID,
	}

	baseReader := kafka.NewReader(readerConfig)
	reader, err := otelkafka.NewReader(baseReader)
	if err != nil {
		return err
	}
	infra.messageConsumer = reader

	// Create Kafka writer
	baseWriter := &kafka.Writer{
		Addr:         kafka.TCP(infra.config.KafkaBroker),
		Topic:        config.InventoryTopic,
		Balancer:     &kafka.LeastBytes{},
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
	infra.messageProducer = writer

	return nil
}

// Shutdown gracefully shuts down all infrastructure components
func (infra *Infrastructure) Shutdown(ctx context.Context) {
	infra.logger.Info("Shutting down infrastructure...")

	// Close Kafka components
	if infra.messageConsumer != nil {
		if err := infra.messageConsumer.Close(); err != nil {
			infra.logger.Error("Failed to close message consumer", zap.Error(err))
		}
	}

	if infra.messageProducer != nil {
		if err := infra.messageProducer.Close(); err != nil {
			infra.logger.Error("Failed to close message producer", zap.Error(err))
		}
	}

	// Shutdown OpenTelemetry
	if infra.otelTraceShutdown != nil {
		if err := infra.otelTraceShutdown(ctx); err != nil {
			infra.logger.Error("Failed to shutdown OTel tracing", zap.Error(err))
		}
	}

	if infra.otelLogShutdown != nil {
		if err := infra.otelLogShutdown(ctx); err != nil {
			infra.logger.Error("Failed to shutdown OTel logging", zap.Error(err))
		}
	}

	// Sync logger
	if err := infra.logger.Sync(); err != nil {
		// Can't log this error since logger might be closed
		fmt.Printf("Failed to sync logger: %v\n", err)
	}
}

// Getters for accessing infrastructure components
func (infra *Infrastructure) Logger() infra_obs.Logger { return infra.logger }
func (infra *Infrastructure) Tracer() infra_obs.Tracer { return infra.tracer }
func (infra *Infrastructure) MessageConsumer() messaging.MessageConsumer {
	return infra.messageConsumer
}
func (infra *Infrastructure) MessageProducer() messaging.MessageProducer {
	return infra.messageProducer
}
