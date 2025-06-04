# Distributed Tracing Demo with OpenTelemetry

A complete distributed tracing demo showing **Java Order Service** → **Kafka** → **Go Inventory Service** with end-to-end tracing using OpenTelemetry and Grafana Cloud.

## 🚀 **Two Operational Modes**

### 🏠 **Mode 1: Hybrid Development (Recommended)**
- **Services**: Run locally (for debugging/development)
- **Infrastructure**: Run in Docker (Kafka, PostgreSQL, Zookeeper)

```bash
# Start infrastructure only
docker compose up -d

# Run services locally (in separate terminals)
cd services/order-service && mvn spring-boot:run
cd services/inventory-service && go run .

# Or use IDE debug configurations (IntelliJ/VS Code)
```

### 🐳 **Mode 2: Full Docker**
- **Everything**: Runs in Docker containers

```bash
# Start everything in Docker
docker compose -f docker-compose.full.yml up --build

# Test the setup
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{"productId": "123", "quantity": 2}'
```

## 📁 **Project Structure**

```
distributed-tracing-otel-demo/
├── docker-compose.yml           # Default: Infrastructure only
├── docker-compose.full.yml      # Full stack in Docker  
├── docker-compose.infra.yml     # Explicit infrastructure only
├── services/
│   ├── order-service/           # Java Spring Boot
│   │   ├── .env                 # Local development config
│   │   ├── .env.example         # Template
│   │   └── README.md            # Service-specific docs
│   └── inventory-service/       # Go service
│       ├── .env                 # Local development config
│       ├── .env.example         # Template
│       └── README.md            # Service-specific docs
└── .vscode/launch.json          # Debug configurations
```

## ⚙️ **Configuration**

All configuration uses **environment variables**:

- **Local development**: Uses `.env` files (connects to `localhost`)
- **Docker deployment**: Uses container environment variables (connects to internal Docker network)

### Quick Setup

```bash
# Copy environment templates
cp services/order-service/.env.example services/order-service/.env
cp services/inventory-service/.env.example services/inventory-service/.env

# Update with your Grafana Cloud credentials in both .env files
```

## 🔍 **Tracing Flow**

1. **HTTP Request** → Order Service
2. **Database Query** → PostgreSQL  
3. **Kafka Message** → Published to `order-events` topic
4. **Kafka Consumer** → Inventory Service processes message
5. **All operations traced** → Grafana Cloud

## 🛠 **Development Workflow**

### Daily Development (Hybrid Mode)
```bash
# 1. Start infrastructure
docker compose up -d

# 2. Run services in debug mode (IDE or terminal)
# Order Service: IntelliJ run configuration or mvn spring-boot:run
# Inventory Service: VS Code debug or go run .

# 3. Develop, debug, and test
curl -X POST http://localhost:8080/api/orders -H "Content-Type: application/json" -d '{"productId": "123", "quantity": 2}'

# 4. Stop infrastructure when done
docker compose down
```

### Integration Testing (Full Docker)
```bash
# Build and test everything together
docker compose -f docker-compose.full.yml up --build

# Test and verify tracing
curl -X POST http://localhost:8080/api/orders -H "Content-Type: application/json" -d '{"productId": "123", "quantity": 2}'

# Check Grafana Cloud for traces
docker compose -f docker-compose.full.yml down
```

## 🎯 **Available Services**

- **Order Service**: `http://localhost:8080` (Java Spring Boot)
- **Inventory Service**: Runs as Kafka consumer (Go)
- **PostgreSQL**: `localhost:5432` (user: `postgres`, password: `password`)
- **Kafka**: `localhost:9092`
- **Database UI**: `http://localhost:8000` (postgres-mcp)

## 📊 **Observability**

- **Traces**: Grafana Cloud OTLP endpoint
- **Logs**: Application logs with trace context
- **Metrics**: Basic service metrics (can be enabled)

## 🚨 **Troubleshooting**

### Services can't connect to infrastructure
```bash
# Make sure infrastructure is running
docker compose ps

# Check if ports are available
lsof -i :5432  # PostgreSQL
lsof -i :9092  # Kafka
```

### Tracing not working
- Verify Grafana Cloud credentials in `.env` files
- Check service logs for OTLP connection errors
- Ensure `OTEL_*` environment variables are loaded

### Docker issues
```bash
# Clean up everything
docker compose down --volumes --remove-orphans
docker compose -f docker-compose.full.yml down --volumes --remove-orphans

# Rebuild from scratch
docker compose build --no-cache
```
