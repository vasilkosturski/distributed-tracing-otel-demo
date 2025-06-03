package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	fmt.Println("🔍 Testing OpenTelemetry Authentication with Grafana Cloud...")

	// Read environment variables (using the exact format from Grafana dashboard)
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	headers := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")
	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")

	fmt.Printf("📡 Endpoint: %s\n", endpoint)
	fmt.Printf("🔐 Headers: %s\n", headers)
	fmt.Printf("📋 Protocol: %s\n", protocol)

	if endpoint == "" || headers == "" {
		log.Fatal("❌ Missing required environment variables. Please set OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_EXPORTER_OTLP_HEADERS")
	}

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

	// Parse headers from environment variable
	headersMap := make(map[string]string)
	if headers != "" {
		// Headers format: "Authorization=Basic TOKEN"
		// Simple parsing for this test
		if len(headers) > 13 && headers[:13] == "Authorization" {
			headersMap["Authorization"] = headers[14:] // Skip "Authorization="
		}
	}

	fmt.Printf("🔑 Parsed Authorization: %s...\n", headersMap["Authorization"][:20])

	// Clean endpoint - remove https:// if present since WithEndpoint doesn't expect it
	cleanEndpoint := endpoint
	if strings.HasPrefix(endpoint, "https://") {
		cleanEndpoint = strings.TrimPrefix(endpoint, "https://")
	} else if strings.HasPrefix(endpoint, "http://") {
		cleanEndpoint = strings.TrimPrefix(endpoint, "http://")
	}

	fmt.Printf("🌐 Clean Endpoint: %s\n", cleanEndpoint)

	// Create trace exporter with proper configuration
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cleanEndpoint),
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
	_, span := tracer.Start(ctx, "test-authentication")
	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.url", "https://example.com/test"),
		attribute.String("test.type", "authentication"),
	)

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	span.End()

	fmt.Println("📤 Test span created and ended, waiting for export...")

	// Give time for the span to be exported
	time.Sleep(3 * time.Second)

	fmt.Println("✅ Test completed! Check your Grafana dashboard for the trace.")
}
