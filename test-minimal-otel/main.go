package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	fmt.Println("🔍 Testing OpenTelemetry with HARDCODED credentials...")

	// HARDCODED values from the working .env files - matching inventory service exactly
	endpoint := "otlp-gateway-prod-eu-west-2.grafana.net"
	authHeader := "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YjNSc2NDMTBiMnRsYmkweUlpd2lheUk2SW1ad2FXMWplRUV3Tnpnek9URXplRFZ5YWpoMlpWa3lkeUlzSW0waU9uc2ljaUk2SW5CeWIyUXRaWFV0ZDJWemRDMHlJbjE5" // Full header with "Basic " prefix

	fmt.Printf("📡 Endpoint: https://%s\n", endpoint)
	fmt.Printf("🔐 Auth Header: %s...\n", authHeader[:30])

	// Set up OpenTelemetry
	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("test-minimal-otel"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create resource: %v", err)
	}

	// Create headers map - EXACTLY like inventory service
	headersMap := map[string]string{
		"Authorization": authHeader, // Already includes "Basic " prefix
	}

	fmt.Printf("🔑 Authorization header: %s...\n", authHeader[:30])

	// Create trace exporter - EXACTLY like inventory service (just hostname)
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint), // Just hostname, no https://
		otlptracehttp.WithURLPath("/otlp/v1/traces"),
		otlptracehttp.WithHeaders(headersMap),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create trace exporter: %v", err)
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	defer func() {
		fmt.Println("🔄 Shutting down tracer provider...")
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("⚠️  Error shutting down tracer provider: %v", err)
		}
	}()

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Create a tracer
	tracer := otel.Tracer("test-minimal-otel")

	fmt.Println("✅ OpenTelemetry setup complete, creating test span...")

	// Create a test span
	_, span := tracer.Start(ctx, "test-hardcoded-auth")
	span.SetAttributes(
		attribute.String("test.type", "hardcoded-credentials"),
		attribute.String("test.endpoint", endpoint),
		attribute.String("test.environment", "local"),
		attribute.Bool("test.success", true),
	)

	// Simulate some work
	fmt.Println("⏳ Simulating some work...")
	time.Sleep(100 * time.Millisecond)

	span.End()
	fmt.Println("📤 Test span created and ended, waiting for export...")

	// Give time for the span to be exported
	time.Sleep(5 * time.Second)

	fmt.Println("✅ Test completed! If successful, check your Grafana dashboard for the trace 'test-hardcoded-auth'")
}
