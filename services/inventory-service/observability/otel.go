package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"inventoryservice/config"

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

// SetupLoggingSDK configures OpenTelemetry SDK specifically for LOGS.
func SetupLoggingSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
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
	handleErr("OTLP Log Exporter", errExporter)

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
	}

	return shutdown, currentErr // Return accumulated errors
}

// SetupTracingSDK configures OpenTelemetry SDK specifically for TRACES.
func SetupTracingSDK(ctx context.Context) (tp *sdktrace.TracerProvider, shutdown func(context.Context) error, err error) {
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

		// Set the global tracer provider
		otel.SetTracerProvider(tp)

		addShutdown("TracerProvider", tp.Shutdown)
	}

	return tp, shutdown, currentErr // Return accumulated errors
}
