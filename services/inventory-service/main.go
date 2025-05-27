package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"inventoryservice/config"
	stdlog "log" // Alias standard log to prevent conflict
	"os"
	"os/signal"
	"time"

	// Local config package

	otelkafka "github.com/Trendyol/otel-kafka-konsumer" // OpenTelemetry Kafka instrumentation
	"github.com/segmentio/kafka-go"

	"go.opentelemetry.io/contrib/bridges/otelzap" // Official OTel Contrib bridge for Zap
	"go.opentelemetry.io/otel"                    // OpenTelemetry API
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"     // OTLP HTTP for Logs
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp" // OTLP HTTP for Traces
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"      // SDK for traces
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0" // Add this for SpanFromContext
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	AppLogger *zap.Logger // Global ZapLogger
)

type OrderCreatedEvent struct {
	OrderID string `json:"order_id"`
}

type InventoryReservedEvent struct {
	OrderID string `json:"order_id"`
}

// setupLoggingOTelSDK configures OpenTelemetry SDK specifically for LOGS.
func setupLoggingOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error
	var currentErr error // To accumulate errors from various setup steps

	shutdown = func(ctx context.Context) error {
		var errs error
		// LIFO order for shutdown
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			errs = errors.Join(errs, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		return errs
	}

	addShutdown := func(name string, fn func(context.Context) error) {
		if fn != nil {
			shutdownFuncs = append(shutdownFuncs, fn)
		}
	}
	// handleErr accumulates errors. It's called after each setup step that can fail.
	handleErr := func(componentName string, inErr error) {
		if inErr != nil {
			currentErr = errors.Join(currentErr, fmt.Errorf("failed to setup %s: %w", componentName, inErr))
		}
	}

	// 1. Build Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil { // If resource creation fails, we can't proceed
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// --- Common OTLP/HTTP Exporter Options ---
	grafanaAuthHeader := map[string]string{"Authorization": config.GrafanaAuthHeader}
	grafanaBaseEndpoint := config.GrafanaBaseEndpoint

	// 2. Setup Logger Provider using OTLP/HTTP (for Zap bridge)
	logExporter, errExporter := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(grafanaBaseEndpoint),
		otlploghttp.WithURLPath(config.LogsPath),
		otlploghttp.WithHeaders(grafanaAuthHeader),
	)
	handleErr("OTLP Log Exporter", errExporter) // Pass errExporter here

	// Proceed only if exporter was created successfully
	if errExporter == nil {
		// Configure BatchProcessor with specific options
		logProcessor := sdklog.NewBatchProcessor(logExporter,
			sdklog.WithExportTimeout(30*time.Second), // SDK default is 30s
			sdklog.WithMaxQueueSize(2048),            // SDK default
		)

		lp := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(logProcessor),
		)
		global.SetLoggerProvider(lp) // Set for otelzap bridge
		addShutdown("LoggerProvider", lp.Shutdown)
		// AppLogger might not be initialized yet when this is first called.
		// Consider logging this success message after AppLogger is fully re-initialized in main().
		// fmt.Println("✅ OTel LoggerProvider initialized for inventory-service.")
	} else {
		// If exporter failed, we might not want to proceed with setting up the LoggerProvider
		// or we set a no-op one. For now, currentErr will have the exporter error.
	}

	return shutdown, currentErr // Return accumulated errors
}

// setupTracingOTelSDK configures OpenTelemetry SDK specifically for TRACES.
func setupTracingOTelSDK(ctx context.Context) (tp *sdktrace.TracerProvider, shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error
	var currentErr error // To accumulate errors from various setup steps

	shutdown = func(ctx context.Context) error {
		var errs error
		// LIFO order for shutdown
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			errs = errors.Join(errs, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		return errs
	}

	addShutdown := func(name string, fn func(context.Context) error) {
		if fn != nil {
			shutdownFuncs = append(shutdownFuncs, fn)
		}
	}
	// handleErr accumulates errors. It's called after each setup step that can fail.
	handleErr := func(componentName string, inErr error) {
		if inErr != nil {
			currentErr = errors.Join(currentErr, fmt.Errorf("failed to setup %s: %w", componentName, inErr))
		}
	}

	// 1. Build Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil { // If resource creation fails, we can't proceed
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up context propagation for distributed tracing
	// This enables trace context to be properly propagated in Kafka headers
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// --- Common OTLP/HTTP Exporter Options ---
	grafanaAuthHeader := map[string]string{"Authorization": config.GrafanaAuthHeader}
	grafanaBaseEndpoint := config.GrafanaBaseEndpoint

	// 2. Setup Trace Exporter using OTLP/HTTP
	traceExporter, errExporter := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(grafanaBaseEndpoint),
		otlptracehttp.WithURLPath(config.TracesPath),
		otlptracehttp.WithHeaders(grafanaAuthHeader),
	)
	handleErr("OTLP Trace Exporter", errExporter)

	// Proceed only if exporter was created successfully
	if errExporter == nil {
		// Configure TracerProvider with batch span processor
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter,
				sdktrace.WithMaxQueueSize(2048),
				sdktrace.WithBatchTimeout(5*time.Second),
			),
			sdktrace.WithResource(res),
		)

		// MOVE this line to AFTER fully initializing tp
		otel.SetTracerProvider(tp)

		addShutdown("TracerProvider", tp.Shutdown)
	} else {
		// If exporter failed, we might not want to proceed with setting up the TracerProvider
		// or we set a no-op one. For now, currentErr will have the exporter error.
	}

	return tp, shutdown, currentErr // Return accumulated errors
}

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
	otelLogShutdown, otelLogSetupErr := setupLoggingOTelSDK(ctx)
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
	tp, otelTraceShutdown, otelTraceSetupErr := setupTracingOTelSDK(ctx)
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

		// Extract trace context from the incoming Kafka message headers
		// This is important for connecting the trace from the producer
		propagator := otel.GetTextMapPropagator()
		carrier := propagation.MapCarrier{}

		// Extract headers from Kafka message to our carrier
		for _, header := range msg.Headers {
			carrier[string(header.Key)] = string(header.Value)
		}

		// Extract the context from the carrier - this will have the parent span info
		msgCtx := propagator.Extract(ctx, carrier)

		AppLogger.Info("📨 Raw Kafka message received",
			zap.ByteString("key", msg.Key),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)

		var order OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			AppLogger.Error("❌ Invalid JSON in OrderCreated event",
				zap.Error(err),
				zap.ByteString("raw_value", msg.Value),
			)
			continue
		}

		AppLogger.Info("✅ Received OrderCreated event processed", zap.String("order_id", order.OrderID))

		// Create a custom span for inventory checking to demonstrate manual tracing
		tracer := otel.Tracer(config.ServiceName)
		_, span := tracer.Start(msgCtx, "inventory_check")

		// Add custom attributes to the span
		span.SetAttributes(
			attribute.String("order.id", order.OrderID),
			attribute.String("inventory.operation", "stock_check"),
			attribute.String("service.component", "inventory_manager"),
		)

		// Simulate inventory check (in real system this would check stock levels)
		AppLogger.Info("🔍 Checking inventory for order", zap.String("order_id", order.OrderID))

		// Simulate some inventory lookup logic with realistic attributes
		time.Sleep(50 * time.Millisecond) // Simulate inventory lookup time

		// In a real system, you'd query a database or external service here
		// For demo purposes, we'll simulate successful inventory check
		stockAvailable := true
		reservedQuantity := 1

		// Add more business-specific attributes
		span.SetAttributes(
			attribute.Bool("inventory.available", stockAvailable),
			attribute.Int("inventory.reserved_quantity", reservedQuantity),
			attribute.String("inventory.status", "reserved"),
		)

		// Mark the span as successful
		span.SetStatus(codes.Ok, "Inventory successfully reserved")

		// End the custom span
		span.End()

		out := InventoryReservedEvent(order)
		payload, err := json.Marshal(out)
		if err != nil {
			AppLogger.Error("❌ Failed to serialize InventoryReserved event",
				zap.Error(err),
				zap.String("order_id", order.OrderID),
			)
			continue
		}

		// Create a message with context that will propagate the trace
		// Using WriteMessage (singular) instead of WriteMessages (plural) for proper tracing
		// See: https://github.com/Trendyol/otel-kafka-konsumer/issues/4
		kafkaMsg := kafka.Message{
			Value: payload,
			Key:   []byte(order.OrderID),
		}
		err = writer.WriteMessage(msgCtx, kafkaMsg)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				AppLogger.Info("Context done, aborting Kafka write.", zap.Error(err))
				break
			}
			AppLogger.Error("❌ Failed to publish InventoryReserved event",
				zap.Error(err),
				zap.String("order_id", order.OrderID),
			)
		} else {
			AppLogger.Info("📤 Sent InventoryReserved event", zap.String("order_id", order.OrderID))
		}
	}

	AppLogger.Info("Inventory Service event loop finished. Shutting down...")
}
