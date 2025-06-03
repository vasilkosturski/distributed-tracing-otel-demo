#!/bin/bash

echo "🚀 Setting up OpenTelemetry Environment Variables from Grafana Dashboard..."

# Using the CORRECT base64 encoding from the raw API key (not the pre-encoded one)
export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp-gateway-prod-eu-west-2.grafana.net"
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Basic MTE5NzE2NzpnbGNfZXlKdklqb2lNVE0zTXpVM09DSXNJbTRpT2lKemRHRmpheTF4TVRrM01UWTNMVzkwYkhBdGQzSnBkR1V0YjNSc2NDMTBiMnRsYmkweUlpd2lheUk2SW1ad2FXMWplRUV3Tnpnek9URXplRFZ5YWpoMlpWa3lkeUlzSW0waU9uc2ljaUk2SW5CeWIyUXRaWFV0ZDJWemRDMHlJbjE5"

echo "✅ Environment variables set:"
echo "   OTEL_EXPORTER_OTLP_PROTOCOL: $OTEL_EXPORTER_OTLP_PROTOCOL"
echo "   OTEL_EXPORTER_OTLP_ENDPOINT: $OTEL_EXPORTER_OTLP_ENDPOINT" 
echo "   OTEL_EXPORTER_OTLP_HEADERS: ${OTEL_EXPORTER_OTLP_HEADERS:0:30}..."

echo ""
echo "🔧 Initializing Go module..."
go mod init test-minimal-otel 2>/dev/null || echo "   Module already initialized"

echo ""
echo "📦 Downloading dependencies..."
go mod tidy

echo ""
echo "🧪 Running minimal OpenTelemetry authentication test..."
go run main.go 