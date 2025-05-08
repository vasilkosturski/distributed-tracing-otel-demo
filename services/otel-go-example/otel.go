package main

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Shared resource (consistent service name)
	res, err := newResource()
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(newPropagator())

	// Traces
	tracerProvider, err := newTracerProvider(res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Metrics
	meterProvider, err := newMeterProvider(res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Logs
	loggerProvider, err := newLoggerProvider(res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newResource() (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("otel-go-example"),
		),
	)
}

func newTracerProvider(res *resource.Resource) (*trace.TracerProvider, error) {
	ctx := context.Background()

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("otlp-gateway-prod-eu-west-2.grafana.net"),
		otlptracehttp.WithURLPath("/otlp/v1/traces"),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YlhrdGIzUnNjQzFoWTJObGMzTXRkRzlyWlc0dE1pSXNJbXNpT2lKS01teDNWVEkzYkcwd01IbzJNVEpGU0RoUFZUQnVjVllpTENKdElqcDdJbklpT2lKd2NtOWtMV1YxTFhkbGMzUXRNaUo5ZlE9PQ==",
		}),
	)
	if err != nil {
		return nil, err
	}

	return trace.NewTracerProvider(
		trace.WithBatcher(traceExporter, trace.WithBatchTimeout(1*time.Second)),
		trace.WithResource(res),
	), nil
}

func newMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	ctx := context.Background()

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint("otlp-gateway-prod-eu-west-2.grafana.net"),
		otlpmetrichttp.WithURLPath("/otlp/v1/metrics"),
		otlpmetrichttp.WithHeaders(map[string]string{
			"Authorization": "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YlhrdGIzUnNjQzFoWTJObGMzTXRkRzlyWlc0dE1pSXNJbXNpT2lKS01teDNWVEkzYkcwd01IbzJNVEpGU0RoUFZUQnVjVllpTENKdElqcDdJbklpT2lKd2NtOWtMV1YxTFhkbGMzUXRNaUo5ZlE9PQ==",
		}),
	)
	if err != nil {
		return nil, err
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(3*time.Second))),
		sdkmetric.WithResource(res),
	), nil
}

func newLoggerProvider(res *resource.Resource) (*sdklog.LoggerProvider, error) {
	ctx := context.Background()

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint("otlp-gateway-prod-eu-west-2.grafana.net"),
		otlploghttp.WithURLPath("/otlp/v1/logs"),
		otlploghttp.WithHeaders(map[string]string{
			"Authorization": "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YlhrdGIzUnNjQzFoWTJObGMzTXRkRzlyWlc0dE1pSXNJbXNpT2lKS01teDNWVEkzYkcwd01IbzJNVEpGU0RoUFZUQnVjVllpTENKdElqcDdJbklpT2lKd2NtOWtMV1YxTFhkbGMzUXRNaUo5ZlE9PQ==",
		}),
	)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	), nil
}
