package main

import (
	"context"
	"fmt"
	"os"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	// 1. Build Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("shipping-service-test"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to build resource: %v\n", err)
		os.Exit(1)
	}

	// 2. Create OTLP log exporter to Grafana Cloud
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint("otlp-gateway-prod-eu-west-2.grafana.net:4317"),
		otlploggrpc.WithHeaders(map[string]string{
			"Authorization": "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YlhrdGIzUnNjQzFoWTJObGMzTXRkRzlyWlc0dE1pSXNJbXNpT2lKS01teDNWVEkzYkcwd01IbzJNVEpGU0RoUFZUQnVjVllpTENKdElqcDdJbklpT2lKd2NtOWtMV1YxTFhkbGMzUXRNaUo5ZlE9PQ==",
		}),
		otlploggrpc.WithTLSCredentials(nil), // enables secure connection
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create OTLP exporter: %v\n", err)
		os.Exit(1)
	}

	// 3. LoggerProvider setup
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	defer loggerProvider.Shutdown(ctx)

	// 4. Register global logger provider
	global.SetLoggerProvider(loggerProvider)

	// 5. Create zap logger and bridge to OpenTelemetry
	baseLogger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("failed to create zap logger: %w", err))
	}
	defer baseLogger.Sync()

	otelLogger := otelzap.New(baseLogger)
	defer otelLogger.Sync()

	log := otelLogger.Ctx(ctx)

	// ✅ Emit a test log
	log.Info("🚀 Minimal test log from otelzap → Grafana Cloud!")

	fmt.Println("✅ Done — check Grafana Cloud logs now")
}
