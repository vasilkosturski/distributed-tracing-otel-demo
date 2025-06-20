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

type BaseSDK struct {
	shutdownFuncs []func(context.Context) error
}

func (sdk *BaseSDK) Close(ctx context.Context) error {
	var err error
	for _, fn := range sdk.shutdownFuncs {
		err = errors.Join(err, fn(ctx))
	}
	sdk.shutdownFuncs = nil
	return err
}

type LoggingSDK struct {
	BaseSDK
	loggerProvider *sdklog.LoggerProvider
}

func (sdk *LoggingSDK) LoggerProvider() *sdklog.LoggerProvider {
	return sdk.loggerProvider
}

type TracingSDK struct {
	BaseSDK
	tracerProvider *sdktrace.TracerProvider
}

func (sdk *TracingSDK) TracerProvider() *sdktrace.TracerProvider {
	return sdk.tracerProvider
}

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

func otlpEndpointConfig(cfg *config.Config, path string) map[string]string {
	return map[string]string{
		"Authorization": cfg.OtelAuthHeader,
	}
}

func handleSetupError(name string, err error, currentErr error) error {
	if err != nil {
		return errors.Join(currentErr, fmt.Errorf("%s: %w", name, err))
	}
	return currentErr
}

func SetupLoggingSDK(ctx context.Context, cfg *config.Config) (*LoggingSDK, error) {
	sdk := &LoggingSDK{}
	var currentErr error

	res, err := createResource()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	logExporter, errExporter := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(cfg.OtelEndpoint),
		otlploghttp.WithURLPath(config.LogsPath),
		otlploghttp.WithHeaders(otlpEndpointConfig(cfg, config.LogsPath)),
	)
	currentErr = handleSetupError("OTLP Log Exporter", errExporter, currentErr)

	if errExporter == nil {
		logProcessor := sdklog.NewBatchProcessor(logExporter,
			sdklog.WithExportTimeout(config.ExportTimeout),
			sdklog.WithMaxQueueSize(config.MaxQueueSize),
		)

		sdk.loggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithProcessor(logProcessor),
			sdklog.WithResource(res),
		)

		sdk.shutdownFuncs = append(sdk.shutdownFuncs, sdk.loggerProvider.Shutdown)
	}

	return sdk, currentErr
}

func SetupTracingSDK(ctx context.Context, cfg *config.Config) (*TracingSDK, error) {
	sdk := &TracingSDK{}
	var currentErr error

	res, err := createResource()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	traceExporter, errExporter := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OtelEndpoint),
		otlptracehttp.WithURLPath(config.TracesPath),
		otlptracehttp.WithHeaders(otlpEndpointConfig(cfg, config.TracesPath)),
	)
	currentErr = handleSetupError("OTLP Trace Exporter", errExporter, currentErr)

	if errExporter == nil {
		traceProcessor := sdktrace.NewBatchSpanProcessor(traceExporter,
			sdktrace.WithExportTimeout(config.ExportTimeout),
			sdktrace.WithMaxQueueSize(config.MaxQueueSize),
		)

		sdk.tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(traceProcessor),
		)

		otel.SetTracerProvider(sdk.tracerProvider)

		sdk.shutdownFuncs = append(sdk.shutdownFuncs, sdk.tracerProvider.Shutdown)
	}

	return sdk, currentErr
}
