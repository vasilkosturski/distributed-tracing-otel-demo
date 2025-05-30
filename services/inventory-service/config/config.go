package config

import (
	"fmt"
	"os"
	"time"
)

// Service configuration constants
const (
	ServiceName    = "inventory-service"
	ServiceVersion = "0.1.0"
)

// Kafka configuration constants
const (
	OrderCreatedTopic = "OrderCreated"
	InventoryTopic    = "InventoryReserved"
	GroupID           = "inventory-service-group"
	BatchTimeout      = 10 * time.Millisecond
	BatchSize         = 100
)

// OpenTelemetry configuration constants
const (
	LogsPath      = "/otlp/v1/logs"   // Grafana Cloud OTLP path
	TracesPath    = "/otlp/v1/traces" // Grafana Cloud OTLP path
	ExportTimeout = 30 * time.Second
	MaxQueueSize  = 2048
)

// Config holds environment-specific configuration
type Config struct {
	// Only the things that change between environments
	KafkaBroker    string
	OtelEndpoint   string
	OtelAuthHeader string
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() (*Config, error) {
	config := &Config{
		KafkaBroker:    getEnvOrDefault("KAFKA_BROKER", "localhost:9092"),
		OtelEndpoint:   getEnvOrDefault("OTEL_ENDPOINT", "otlp-gateway-prod-eu-west-2.grafana.net"),
		OtelAuthHeader: os.Getenv("OTEL_AUTH_HEADER"), // Required for Grafana
	}

	// Basic validation
	if config.KafkaBroker == "" {
		return nil, fmt.Errorf("KAFKA_BROKER cannot be empty")
	}
	if config.OtelEndpoint == "" {
		return nil, fmt.Errorf("OTEL_ENDPOINT cannot be empty")
	}
	if config.OtelAuthHeader == "" {
		return nil, fmt.Errorf("OTEL_AUTH_HEADER is required for Grafana Cloud")
	}

	return config, nil
}

// Helper function
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
