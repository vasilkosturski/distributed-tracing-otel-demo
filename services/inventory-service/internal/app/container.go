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
	config          *config.Config
	logger          observability.Logger
	tracer          observability.Tracer
	messageConsumer kafka.Consumer
	messageProducer kafka.Producer
	shutdownFunc    func(context.Context)
}

// NewContainer creates and initializes all infrastructure components
func NewContainer(ctx context.Context) (*Container, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logging infrastructure
	basicLogger, err := createBasicLogger()
	if err != nil {
		return nil, err
	}

	enhancedLogger, otelLogShutdown, err := setupOTelLogging(ctx, cfg, basicLogger)
	if err != nil {
		return nil, err
	}

	// Setup tracing infrastructure
	tp, otelTraceShutdown, err := setupOTelTracing(ctx, cfg, enhancedLogger)
	if err != nil {
		return nil, err
	}

	// Setup Kafka infrastructure
	messageConsumer, messageProducer, err := setupKafka(cfg, tp)
	if err != nil {
		return nil, err
	}

	// Create shutdown function that captures cleanup functions
	shutdownFunc := func(ctx context.Context) {
		enhancedLogger.Info("Shutting down infrastructure...")

		if messageConsumer != nil {
			if err := messageConsumer.Close(); err != nil {
				enhancedLogger.Error("Failed to close message consumer", zap.Error(err))
			}
		}

		if messageProducer != nil {
			if err := messageProducer.Close(); err != nil {
				enhancedLogger.Error("Failed to close message producer", zap.Error(err))
			}
		}

		if otelTraceShutdown != nil {
			if err := otelTraceShutdown(ctx); err != nil {
				enhancedLogger.Error("Failed to shutdown OTel tracing", zap.Error(err))
			}
		}

		if otelLogShutdown != nil {
			if err := otelLogShutdown(ctx); err != nil {
				enhancedLogger.Error("Failed to shutdown OTel logging", zap.Error(err))
			}
		}

		if enhancedLogger != nil {
			_ = enhancedLogger.Sync()
		}

		enhancedLogger.Info("Infrastructure shutdown complete")
	}

	container := &Container{
		config:          cfg,
		logger:          enhancedLogger,
		tracer:          tp.Tracer(config.ServiceName),
		messageConsumer: messageConsumer,
		messageProducer: messageProducer,
		shutdownFunc:    shutdownFunc,
	}

	return container, nil
}

func createBasicLogger() (observability.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

func setupOTelLogging(ctx context.Context, cfg *config.Config, basicLogger observability.Logger) (observability.Logger, func(context.Context) error, error) {
	otelLogShutdown, err := observability.SetupLoggingSDK(ctx, cfg)
	if err != nil {
		basicLogger.Error("Failed to setup OpenTelemetry logging", zap.Error(err))
	}

	// Create enhanced logger with OTel integration
	logProvider := global.GetLoggerProvider()
	instrumentationScopeName := "inventory-service"
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
	enhancedLogger := zap.New(finalCore,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.String("service.name", config.ServiceName)),
	)

	enhancedLogger.Info("Logger initialized with OpenTelemetry integration")
	return enhancedLogger, otelLogShutdown, err
}

func setupOTelTracing(ctx context.Context, cfg *config.Config, logger observability.Logger) (trace.TracerProvider, func(context.Context) error, error) {
	tp, otelTraceShutdown, err := observability.SetupTracingSDK(ctx, cfg)
	if err != nil {
		logger.Error("Failed to setup OpenTelemetry tracing", zap.Error(err))
	}
	return tp, otelTraceShutdown, err
}

// setupKafkaWithTracer initializes Kafka consumer and producer with OpenTelemetry
func setupKafka(cfg *config.Config, tp trace.TracerProvider) (kafka.Consumer, kafka.Producer, error) {
	readerConfig := kafkago.ReaderConfig{
		Brokers: []string{cfg.KafkaBroker},
		Topic:   config.OrderCreatedTopic,
		GroupID: config.GroupID,
	}

	baseReader := kafkago.NewReader(readerConfig)
	reader, err := otelkafka.NewReader(baseReader)
	if err != nil {
		return nil, nil, err
	}

	baseWriter := &kafkago.Writer{
		Addr:         kafkago.TCP(cfg.KafkaBroker),
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
		return nil, nil, err
	}

	return reader, writer, nil
}

// Shutdown gracefully shuts down all infrastructure components
func (c *Container) Shutdown(ctx context.Context) {
	c.shutdownFunc(ctx)
}

// Getters for accessing infrastructure components
func (c *Container) Logger() observability.Logger    { return c.logger }
func (c *Container) Tracer() observability.Tracer    { return c.tracer }
func (c *Container) MessageConsumer() kafka.Consumer { return c.messageConsumer }
func (c *Container) MessageProducer() kafka.Producer { return c.messageProducer }
