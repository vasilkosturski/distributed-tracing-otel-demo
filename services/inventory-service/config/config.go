package config

// Kafka configuration
const (
	KafkaBrokerAddress = "localhost:9092"
	OrderCreatedTopic  = "OrderCreated"
	InventoryTopic     = "InventoryReserved"
	GroupID            = "inventory-service-group"
)

// Service configuration
const (
	ServiceName    = "inventory-service"
	ServiceVersion = "0.1.0"
)

// Grafana Cloud configuration
const (
	GrafanaAuthHeader   = "Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTB4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YlhrdGIzUnNjQzFoWTJObGMzTXRkRzlyWlc0dE1pSXNJbXNpT2lKS01teDNWVEkzYkcwd01IbzJNVEpGU0RoUFZUQnVjVllpTENKdElqcDdJbklpT2lKd2NtOWtMV1YxTFhkbGMzUXRNaUo5ZlE9PQ=="
	GrafanaBaseEndpoint = "otlp-gateway-prod-eu-west-2.grafana.net"
	LogsPath            = "/otlp/v1/logs"
	TracesPath          = "/otlp/v1/traces"
)
