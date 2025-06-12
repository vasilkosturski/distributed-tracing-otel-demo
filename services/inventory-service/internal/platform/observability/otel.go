package observability

import (
	"context"
	"errors"
	"fmt"
	"inventoryservice/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupLoggingSDK initializes OpenTelemetry logging with the provided configuration
func SetupLoggingSDK(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error
	var currentErr error // To accumulate errors from various setup steps

	shutdown = func(context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(name string, inErr error) {
		if inErr != nil {
			currentErr = errors.Join(currentErr, fmt.Errorf("%s: %w", name, inErr))
		}
	}

	// 1. Setup Resource (contains service metadata)
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

	// 2. Setup Logger Provider using OTLP/HTTP
	logExporter, errExporter := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(cfg.OtelEndpoint),
		otlploghttp.WithURLPath(config.LogsPath),
		otlploghttp.WithHeaders(map[string]string{"Authorization": cfg.OtelAuthHeader}),
	)
	handleErr("OTLP Log Exporter", errExporter)

	// Proceed only if exporter was created successfully
	if errExporter == nil {
		// Configure BatchProcessor with configurable options
		logProcessor := sdklog.NewBatchProcessor(logExporter,
			sdklog.WithExportTimeout(config.ExportTimeout),
			sdklog.WithMaxQueueSize(config.MaxQueueSize),
		)

		// Create the LoggerProvider
		loggerProvider := sdklog.NewLoggerProvider(
			sdklog.WithProcessor(logProcessor),
			sdklog.WithResource(res),
		)

		// Set the global logger provider
		global.SetLoggerProvider(loggerProvider)

		// Add shutdown function
		shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	}

	return shutdown, currentErr
}

// SetupTracingSDK initializes OpenTelemetry tracing with the provided configuration
func SetupTracingSDK(ctx context.Context, cfg *config.Config) (tp *sdktrace.TracerProvider, shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error
	var currentErr error

	shutdown = func(context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(name string, inErr error) {
		if inErr != nil {
			currentErr = errors.Join(currentErr, fmt.Errorf("%s: %w", name, inErr))
		}
	}

	// 1. Setup Resource (contains service metadata)
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up context propagation for distributed tracing
	// This enables trace context to be properly propagated in Kafka headers
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 2. Setup Trace Provider using OTLP/HTTP
	traceExporter, errExporter := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OtelEndpoint),
		otlptracehttp.WithURLPath(config.TracesPath),
		otlptracehttp.WithHeaders(map[string]string{"Authorization": cfg.OtelAuthHeader}),
	)
	handleErr("OTLP Trace Exporter", errExporter)

	// Proceed only if exporter was created successfully
	if errExporter == nil {
		// Configure BatchProcessor
		traceProcessor := sdktrace.NewBatchSpanProcessor(traceExporter,
			sdktrace.WithExportTimeout(config.ExportTimeout),
			sdktrace.WithMaxQueueSize(config.MaxQueueSize),
		)

		// Create the TracerProvider
		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(traceProcessor),
		)

		// Set the global tracer provider
		otel.SetTracerProvider(tracerProvider)

		// Add shutdown function
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		tp = tracerProvider
	}

	return tp, shutdown, currentErr
}
