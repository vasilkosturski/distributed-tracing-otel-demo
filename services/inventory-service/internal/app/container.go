package app

import (
	"context"
	"fmt"
	"inventoryservice/internal/config"
	"inventoryservice/internal/inventory"
	"inventoryservice/internal/platform/kafka"
	"inventoryservice/internal/platform/observability"
	"os"

	otelkafka "github.com/Trendyol/otel-kafka-konsumer"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Container struct {
	config          *config.Config
	logger          *zap.Logger
	tracer          trace.Tracer
	messageProducer kafka.Producer
	consumerService inventory.ConsumerService
	shutdownFuncs   []func(context.Context) error
}

func NewContainer(ctx context.Context) (*Container, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	container := &Container{
		config:        cfg,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	enhancedLogger, loggingSDK, err := setupOTelLogging(ctx, cfg)
	if err != nil {
		return nil, err
	}
	container.logger = enhancedLogger
	if loggingSDK != nil {
		container.shutdownFuncs = append(container.shutdownFuncs, loggingSDK.Close)
	}

	tracingSDK, err := setupOTelTracing(ctx, cfg, enhancedLogger)
	if err != nil {
		return nil, err
	}
	container.tracer = tracingSDK.TracerProvider().Tracer(config.ServiceName)
	if tracingSDK != nil {
		container.shutdownFuncs = append(container.shutdownFuncs, tracingSDK.Close)
	}

	messageConsumer, messageProducer, err := setupKafka(cfg, tracingSDK.TracerProvider())
	if err != nil {
		return nil, err
	}
	container.messageProducer = messageProducer

	container.shutdownFuncs = append(container.shutdownFuncs, func(ctx context.Context) error {
		return messageProducer.Close()
	})

	inventoryService := inventory.NewInventoryService(container.logger, container.tracer)
	messageHandler := inventory.NewMessageHandler(
		inventoryService,
		container.messageProducer,
		container.logger,
	)

	container.consumerService = inventory.NewConsumerService(
		messageConsumer,
		messageHandler,
		container.logger,
	)

	return container, nil
}

func setupOTelLogging(ctx context.Context, cfg *config.Config) (*zap.Logger, *observability.LoggingSDK, error) {
	basicLogger, err := zap.NewProduction()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create basic logger: %w", err)
	}

	loggingSDK, err := observability.SetupLoggingSDK(ctx, cfg)
	if err != nil {
		basicLogger.Error("Failed to setup OpenTelemetry logging", zap.Error(err))
	}

	otelZapCore := otelzap.NewCore("inventory-service",
		otelzap.WithLoggerProvider(loggingSDK.LoggerProvider()),
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
	return enhancedLogger, loggingSDK, err
}

func setupOTelTracing(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*observability.TracingSDK, error) {
	tracingSDK, err := observability.SetupTracingSDK(ctx, cfg)
	if err != nil {
		logger.Error("Failed to setup OpenTelemetry tracing", zap.Error(err))
	}
	return tracingSDK, err
}

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

func (c *Container) Close(ctx context.Context) error {
	c.logger.Info("Shutting down infrastructure...")

	var closeErr error
	for _, shutdownFunc := range c.shutdownFuncs {
		if err := shutdownFunc(ctx); err != nil {
			c.logger.Error("Error during shutdown", zap.Error(err))
			if closeErr == nil {
				closeErr = err
			}
		}
	}

	if c.logger != nil {
		_ = c.logger.Sync()
	}

	c.logger.Info("Infrastructure shutdown complete")
	return closeErr
}

func (c *Container) Logger() *zap.Logger             { return c.logger }
func (c *Container) Tracer() trace.Tracer            { return c.tracer }
func (c *Container) MessageProducer() kafka.Producer { return c.messageProducer }
func (c *Container) ConsumerService() inventory.ConsumerService {
	return c.consumerService
}
