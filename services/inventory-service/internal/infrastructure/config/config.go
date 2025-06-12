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

// LoadConfig loads configuration from environment variables with validation
func LoadConfig() (*Config, error) {
	config := &Config{
		KafkaBroker:    os.Getenv("KAFKA_BROKER"),
		OtelEndpoint:   os.Getenv("OTEL_ENDPOINT"),
		OtelAuthHeader: os.Getenv("OTEL_AUTH_HEADER"),
	}

	// Validate all required configuration
	if config.KafkaBroker == "" {
		return nil, fmt.Errorf("KAFKA_BROKER environment variable is required")
	}
	if config.OtelEndpoint == "" {
		return nil, fmt.Errorf("OTEL_ENDPOINT environment variable is required")
	}
	if config.OtelAuthHeader == "" {
		return nil, fmt.Errorf("OTEL_AUTH_HEADER environment variable is required")
	}

	return config, nil
}
