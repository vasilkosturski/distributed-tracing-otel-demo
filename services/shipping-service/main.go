package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log" // Alias standard log to prevent conflict
	"os"
	"os/signal"
	"time"

	"github.com/segmentio/kafka-go" // Keep Kafka code, but no OTel instrumentation for it yet

	"go.opentelemetry.io/contrib/bridges/otelzap"                 // Official OTel Contrib bridge for Zap
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp" // OTLP HTTP for Logs
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	kafkaBrokerAddress = "localhost:9092"
	orderCreatedTopic  = "OrderCreated"
	packagingTopic     = "PackagingCompleted"
	groupID            = "shipping-service-group"

	AppLogger *zap.Logger // Global ZapLogger
)

type OrderCreatedEvent struct {
	OrderID string `json:"order_id"`
}

type PackagingCompletedEvent struct {
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
			semconv.ServiceName("shipping-service"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil { // If resource creation fails, we can't proceed
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// --- Common OTLP/HTTP Exporter Options ---
	// Ensure this token is the same one working in your Dice Roller app
	grafanaAuthHeader := map[string]string{"Authorization": "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YlhrdGIzUnNjQzFoWTJObGMzTXRkRzlyWlc0dE1pSXNJbXNpT2lKS01teDNWVEkzYkcwd01IbzJNVEpGU0RoUFZUQnVjVllpTENKdElqcDdJbklpT2lKd2NtOWtMV1YxTFhkbGMzUXRNaUo5ZlE9PQ=="}
	grafanaBaseEndpoint := "otlp-gateway-prod-eu-west-2.grafana.net" // Base hostname

	// 2. Setup Logger Provider using OTLP/HTTP (for Zap bridge)
	logExporter, errExporter := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(grafanaBaseEndpoint),
		otlploghttp.WithURLPath("/otlp/v1/logs"), // <<< CORRECTED PATH
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
		// fmt.Println("✅ OTel LoggerProvider initialized for shipping-service.")
	} else {
		// If exporter failed, we might not want to proceed with setting up the LoggerProvider
		// or we set a no-op one. For now, currentErr will have the exporter error.
	}

	return shutdown, currentErr // Return accumulated errors
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
	AppLogger.Info("Shipping Service attempting to start (initial console logger)...")

	otelShutdown, otelSetupErr := setupLoggingOTelSDK(ctx) // Use the log-focused setup
	if otelSetupErr != nil {
		AppLogger.Error("⚠️ Failed to setup OpenTelemetry SDK for logging completely", zap.Error(otelSetupErr))
	} else {
		AppLogger.Info("✅ OpenTelemetry SDK for logging initialized (or partially).")
	}
	if otelShutdown != nil {
		defer func() {
			AppLogger.Info("Shutting down OpenTelemetry SDK for logging...")
			if shutdownErr := otelShutdown(context.Background()); shutdownErr != nil {
				AppLogger.Error("❌ Error during OTel logging SDK shutdown", zap.Error(shutdownErr))
			}
			AppLogger.Info("OTel logging SDK shutdown complete.")
		}()
	}

	// Re-initialize AppLogger with OTel bridge (official contrib) and console tee
	logProvider := global.GetLoggerProvider()             // Will be a no-op if OTel setup failed
	instrumentationScopeName := "shipping-service.manual" // Customize this
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
		zap.Fields(zap.String("service.name", "shipping-service")),
	)
	AppLogger.Info("🚀 Shipping Service Zap logger re-initialized with OTel bridge and console output.")

	// --- Service Logs ---
	AppLogger.Info("Connecting to Kafka", zap.String("broker", kafkaBrokerAddress))
	AppLogger.Info("Configured topics",
		zap.String("consumer_topic", orderCreatedTopic),
		zap.String("producer_topic", packagingTopic),
	)

	// --- Kafka Setup (using non-instrumented clients for now) ---
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBrokerAddress},
		Topic:   orderCreatedTopic,
		GroupID: groupID,
		MaxWait: 1 * time.Second,
	})
	defer func() {
		AppLogger.Info("Closing Kafka reader...")
		if err := reader.Close(); err != nil {
			AppLogger.Error("❌ Failed to close Kafka reader", zap.Error(err))
		}
	}()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokerAddress),
		Topic:    packagingTopic,
		Balancer: &kafka.LeastBytes{},
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
			time.Sleep(1 * time.Second)
			continue
		}

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

		time.Sleep(50 * time.Millisecond)

		out := PackagingCompletedEvent{OrderID: order.OrderID}
		payload, err := json.Marshal(out)
		if err != nil {
			AppLogger.Error("❌ Failed to serialize PackagingCompleted event",
				zap.Error(err),
				zap.String("order_id", order.OrderID),
			)
			continue
		}

		err = writer.WriteMessages(ctx, kafka.Message{Value: payload, Key: []byte(order.OrderID)})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				AppLogger.Info("Context done, aborting Kafka write.", zap.Error(err))
				break
			}
			AppLogger.Error("❌ Failed to publish PackagingCompleted event",
				zap.Error(err),
				zap.String("order_id", order.OrderID),
			)
		} else {
			AppLogger.Info("📤 Sent PackagingCompleted event", zap.String("order_id", order.OrderID))
		}
	}

	AppLogger.Info("Shipping Service event loop finished. Shutting down...")
}
