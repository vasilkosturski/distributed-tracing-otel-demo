# Configuration Guide

The inventory service uses a **simplified configuration approach** with Grafana Cloud observability - most settings are constants, with only essential deployment-specific values configurable via environment variables.

## Constants (Hardcoded)

Most configuration is kept as constants for consistency:

**Service**: `inventory-service` v`0.1.0`  
**Kafka Topics**: `OrderCreated` → `InventoryReserved`  
**Consumer Group**: `inventory-service-group`  
**OTLP Paths**: `/v1/logs`, `/v1/traces`  
**Performance**: 30s timeout, 2048 queue size, 100ms batch  
**Grafana Endpoint**: `https://otlp-gateway-prod-eu-west-2.grafana.net`

## Environment Variables

Only **2 variables** for deployment differences:

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKER` | `localhost:9092` | Kafka broker address |
| `OTEL_AUTH_HEADER` | _(required)_ | Grafana Cloud auth header |

## Deployment Examples

### Local Development
```bash
export KAFKA_BROKER=localhost:9092
export OTEL_AUTH_HEADER="Basic your-encoded-credentials"
go run .
```

### Docker Compose
```bash
export KAFKA_BROKER=kafka:9092
export OTEL_AUTH_HEADER="Basic your-encoded-credentials"
go run .
```

### Production
```bash
export KAFKA_BROKER=prod-kafka.company.com:9092
export OTEL_AUTH_HEADER="Basic your-production-credentials"
go run .
```

## Why This Approach?

- **Simplicity**: Only 2 environment variables
- **Consistency**: Always uses Grafana Cloud for observability
- **Focus**: Only Kafka broker changes between environments
- **Blog-friendly**: Easy to understand and demonstrate 