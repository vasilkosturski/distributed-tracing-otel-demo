package observability

import (
	"context"
	"errors"
	"fmt"
	"inventoryservice/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// BaseSDK holds common fields for both logging and tracing SDKs
type BaseSDK struct {
	shutdownFuncs []func(context.Context) error
}

// Close implements io.Closer
func (sdk *BaseSDK) Close(ctx context.Context) error {
	var err error
	for _, fn := range sdk.shutdownFuncs {
		err = errors.Join(err, fn(ctx))
	}
	sdk.shutdownFuncs = nil
	return err
}

// LoggingSDK holds the OpenTelemetry logging components
type LoggingSDK struct {
	BaseSDK
	loggerProvider *sdklog.LoggerProvider
}

// LoggerProvider returns the OpenTelemetry logger provider
func (sdk *LoggingSDK) LoggerProvider() *sdklog.LoggerProvider {
	return sdk.loggerProvider
}

// TracingSDK holds the OpenTelemetry tracing components
type TracingSDK struct {
	BaseSDK
	tracerProvider *sdktrace.TracerProvider
}

// TracerProvider returns the OpenTelemetry tracer provider
func (sdk *TracingSDK) TracerProvider() *sdktrace.TracerProvider {
	return sdk.tracerProvider
}

// createResource creates a common OpenTelemetry resource with service metadata
func createResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
}

// otlpEndpointConfig returns common OTLP endpoint configuration
func otlpEndpointConfig(cfg *config.Config, path string) map[string]string {
	return map[string]string{
		"Authorization": cfg.OtelAuthHeader,
	}
}

// handleSetupError handles errors during SDK setup
func handleSetupError(name string, err error, currentErr error) error {
	if err != nil {
		return errors.Join(currentErr, fmt.Errorf("%s: %w", name, err))
	}
	return currentErr
}

// SetupLoggingSDK initializes OpenTelemetry logging with the provided configuration
func SetupLoggingSDK(ctx context.Context, cfg *config.Config) (*LoggingSDK, error) {
	sdk := &LoggingSDK{}
	var currentErr error

	// 1. Setup Resource (contains service metadata)
	res, err := createResource()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 2. Setup Logger Provider using OTLP/HTTP
	logExporter, errExporter := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(cfg.OtelEndpoint),
		otlploghttp.WithURLPath(config.LogsPath),
		otlploghttp.WithHeaders(otlpEndpointConfig(cfg, config.LogsPath)),
	)
	currentErr = handleSetupError("OTLP Log Exporter", errExporter, currentErr)

	// Proceed only if exporter was created successfully
	if errExporter == nil {
		// Configure BatchProcessor with configurable options
		logProcessor := sdklog.NewBatchProcessor(logExporter,
			sdklog.WithExportTimeout(config.ExportTimeout),
			sdklog.WithMaxQueueSize(config.MaxQueueSize),
		)

		// Create the LoggerProvider
		sdk.loggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithProcessor(logProcessor),
			sdklog.WithResource(res),
		)

		// Add shutdown function
		sdk.shutdownFuncs = append(sdk.shutdownFuncs, sdk.loggerProvider.Shutdown)
	}

	return sdk, currentErr
}

// SetupTracingSDK initializes OpenTelemetry tracing with the provided configuration
func SetupTracingSDK(ctx context.Context, cfg *config.Config) (*TracingSDK, error) {
	sdk := &TracingSDK{}
	var currentErr error

	// 1. Setup Resource (contains service metadata)
	res, err := createResource()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
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
		otlptracehttp.WithHeaders(otlpEndpointConfig(cfg, config.TracesPath)),
	)
	currentErr = handleSetupError("OTLP Trace Exporter", errExporter, currentErr)

	// Proceed only if exporter was created successfully
	if errExporter == nil {
		// Configure BatchProcessor
		traceProcessor := sdktrace.NewBatchSpanProcessor(traceExporter,
			sdktrace.WithExportTimeout(config.ExportTimeout),
			sdktrace.WithMaxQueueSize(config.MaxQueueSize),
		)

		// Create the TracerProvider
		sdk.tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(traceProcessor),
		)

		// Set the global tracer provider
		otel.SetTracerProvider(sdk.tracerProvider)

		// Add shutdown function
		sdk.shutdownFuncs = append(sdk.shutdownFuncs, sdk.tracerProvider.Shutdown)
	}

	return sdk, currentErr
}
