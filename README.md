# Distributed Tracing Demo with OpenTelemetry

🚀 **A complete tutorial for building a distributed tracing system** with **Java Order Service** → **Kafka** → **Go Inventory Service** using OpenTelemetry and Grafana Cloud.

## 📖 **What You'll Build**

- **Order Service** (Java Spring Boot) - Handles HTTP requests and publishes events
- **Inventory Service** (Go) - Consumes Kafka events and processes inventory  
- **Full Observability** - End-to-end tracing through HTTP → Database → Kafka → Consumer
- **Two Deployment Modes** - Local development or full Docker setup

## 🏗️ **Architecture Overview**

```
[HTTP Request] → [Order Service] → [PostgreSQL] → [Kafka] → [Inventory Service]
       ↓               ↓              ↓           ↓              ↓
    [Traces]        [Traces]       [Traces]    [Traces]      [Traces]
       ↓               ↓              ↓           ↓              ↓
                    [Grafana Cloud OTLP Endpoint]
```

**Tracing Flow:**
1. HTTP request creates a trace span
2. Database operations are automatically traced  
3. Kafka message publishing creates linked spans
4. Kafka consumer continues the trace context
5. All spans are sent to Grafana Cloud for visualization

---

## 🚀 **Quick Start (5 Minutes)**

### **Step 1: Clone and Setup**
```bash
git clone <repository-url>
cd distributed-tracing-otel-demo
```

### **Step 2: Configure Environment**
```bash
# Copy environment templates
cp services/order-service/.env.example services/order-service/.env
cp services/inventory-service/.env.example services/inventory-service/.env
```

### **Step 3: Add Your Grafana Cloud Credentials**
Edit both `.env` files and replace the placeholder values:

**`services/order-service/.env`:**
```bash
# Database Configuration
DATABASE_URL=jdbc:postgresql://localhost:5432/orders_db
DATABASE_USER=postgres
DATABASE_PASSWORD=password

# Kafka Configuration  
KAFKA_BROKER=localhost:9092

# OpenTelemetry Configuration
OTEL_SERVICE_NAME=order-service
OTEL_TRACES_EXPORTER=otlp
OTEL_LOGS_EXPORTER=otlp
OTEL_METRICS_EXPORTER=none
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces
OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/logs
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic YOUR_GRAFANA_CLOUD_AUTH_TOKEN_HERE
OTEL_INSTRUMENTATION_SLF4J_LOGBACK_APPENDER_ENABLED=true
OTEL_LOGS_INCLUDE_TRACE_CONTEXT=true
```

**`services/inventory-service/.env`:**
```bash
# Kafka Configuration
KAFKA_BROKER=localhost:9092

# Grafana Cloud Configuration
OTEL_AUTH_HEADER=Basic YOUR_ENCODED_CREDENTIALS_HERE
```

> **Getting Grafana Cloud Credentials:**
> 1. Sign up at [grafana.com](https://grafana.com)
> 2. Go to "My Account" → "Cloud Portal" → "Configure"  
> 3. Under "OpenTelemetry", copy your instance details
> 4. Create a base64 encoded token: `echo -n "instanceId:token" | base64`

### **Step 4: Choose Your Mode**

**🏠 Mode 1 - Hybrid Development (Recommended)**
```bash
# Start infrastructure only
docker compose up -d

# Run services locally (separate terminals)
cd services/order-service
source .env && mvn spring-boot:run -Dspring-boot.run.jvmArguments="-javaagent:./opentelemetry-javaagent.jar"

cd services/inventory-service  
source .env && go run .
```

**🐳 Mode 2 - Full Docker**
```bash
# Start everything in Docker
docker compose -f docker-compose.full.yml up --build
```

### **Step 5: Test It!**
```bash
# Create an order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "product_id": "987fcdeb-51a2-43d1-9c47-123456789abc", 
    "quantity": 2
  }'

# Get all orders
curl http://localhost:8080/orders
```

### **Step 6: View Traces**
1. Go to your Grafana Cloud instance
2. Navigate to "Explore" → "Tempo"
3. Search for traces with service name `order-service`
4. See the complete request flow! 🎉

---

## 🔧 **Detailed Configuration**

### **Environment Variables Reference**

#### **Order Service Configuration**
| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `jdbc:postgresql://localhost:5432/orders_db` |
| `DATABASE_USER` | Database username | `postgres` |
| `DATABASE_PASSWORD` | Database password | `password` |
| `KAFKA_BROKER` | Kafka broker address | `localhost:9092` |
| `OTEL_SERVICE_NAME` | Service name in traces | `order-service` |
| `OTEL_TRACES_EXPORTER` | Trace exporter type | `otlp` |
| `OTEL_LOGS_EXPORTER` | Log exporter type | `otlp` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Grafana traces endpoint | `https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces` |
| `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` | Grafana logs endpoint | `https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/logs` |
| `OTEL_EXPORTER_OTLP_HEADERS` | Authorization header | `Authorization=Basic <encoded-token>` |

#### **Inventory Service Configuration**
| Variable | Description | Example |
|----------|-------------|---------|
| `KAFKA_BROKER` | Kafka broker address | `localhost:9092` |
| `OTEL_AUTH_HEADER` | Grafana authorization | `Basic <encoded-token>` |

### **Docker Configuration Differences**

When running in Docker mode, the services connect to internal Docker networks:

- `KAFKA_BROKER=kafka:9093` (instead of `localhost:9092`)
- `DATABASE_URL=jdbc:postgresql://postgres:5432/orders_db` (instead of `localhost:5432`)

The `docker-compose.full.yml` automatically handles these differences.

---

## 🛠️ **Development Modes Explained**

### **🏠 Hybrid Development Mode (Recommended)**

**What runs where:**
- **Infrastructure** (PostgreSQL, Kafka, Zookeeper): Docker containers
- **Services** (Order & Inventory): Your local machine

**Benefits:**
- ✅ Full debugging with breakpoints
- ✅ Instant code reloads  
- ✅ IDE integration
- ✅ Easy log viewing
- ✅ Fast development cycle

**Setup:**
```bash
# Start infrastructure
docker compose up -d

# Check infrastructure is running
docker compose ps

# Run Order Service (Terminal 1)
cd services/order-service
source .env
mvn spring-boot:run -Dspring-boot.run.jvmArguments="-javaagent:./opentelemetry-javaagent.jar"

# Run Inventory Service (Terminal 2)  
cd services/inventory-service
source .env
go run .
```

### **🐳 Full Docker Mode**

**What runs where:**
- **Everything**: Docker containers with internal networking

**Benefits:**
- ✅ Production-like environment
- ✅ Consistent across machines
- ✅ Easy CI/CD integration
- ✅ No local dependencies needed

**Setup:**
```bash
# Start everything
docker compose -f docker-compose.full.yml up --build

# View logs
docker compose -f docker-compose.full.yml logs -f order-service
docker compose -f docker-compose.full.yml logs -f inventory-service

# Stop everything
docker compose -f docker-compose.full.yml down
```

---

## 🧪 **API Reference**

### **Order Service Endpoints**

**Base URL:** `http://localhost:8080`

#### **Create Order**
```http
POST /orders
Content-Type: application/json

{
  "customer_id": "123e4567-e89b-12d3-a456-426614174000",
  "product_id": "987fcdeb-51a2-43d1-9c47-123456789abc",
  "quantity": 2
}
```

**Response:**
```json
{
  "order_id": "order-uuid-here",
  "status": "created"
}
```

#### **Get All Orders**
```http
GET /orders
```

**Response:**
```json
[
  {
    "id": "order-uuid",
    "customer_id": "customer-uuid", 
    "product_id": "product-uuid",
    "quantity": 2,
    "status": "created",
    "created_at": "2024-01-01T12:00:00Z"
  }
]
```

### **Testing Examples**

```bash
# Create multiple orders
for i in {1..5}; do
  curl -X POST http://localhost:8080/orders \
    -H "Content-Type: application/json" \
    -d '{
      "customer_id": "123e4567-e89b-12d3-a456-426614174000",
      "product_id": "987fcdeb-51a2-43d1-9c47-123456789abc",
      "quantity": '$i'
    }'
  echo "Created order $i"
done

# Check all orders
curl -s http://localhost:8080/orders | jq '.'
```

---

## 📊 **Infrastructure Services**

### **PostgreSQL Database**
- **Port:** `5432`
- **Database:** `orders_db`
- **User:** `postgres` 
- **Password:** `password`
- **Connection:** `postgresql://postgres:password@localhost:5432/orders_db`

### **Database UI (postgres-mcp)**
- **URL:** `http://localhost:8000`
- **Access:** View database tables and data through web interface

### **Kafka**
- **Port:** `9092` (external), `9093` (internal Docker)
- **Topics:** `order-events` (created automatically)
- **Zookeeper:** `localhost:2181`

### **Service Ports**
- **Order Service:** `8080`
- **PostgreSQL:** `5432`
- **Kafka:** `9092`
- **Zookeeper:** `2181`
- **DB UI:** `8000`

---

## 🔍 **Understanding the Traces**

### **What Gets Traced**

1. **HTTP Requests** - Every API call to Order Service
2. **Database Operations** - SQL queries to PostgreSQL
3. **Kafka Publishing** - Message publishing to `order-events` topic
4. **Kafka Consuming** - Message processing in Inventory Service
5. **Cross-Service Context** - Trace context flows through Kafka headers

### **Trace Structure**

```
Root Span: POST /orders
├── Child Span: Database INSERT
├── Child Span: Kafka PUBLISH
└── Child Span: Inventory Processing (different service)
```

### **Key Trace Attributes**

- **Service Names:** `order-service`, `inventory-service`
- **Operation Names:** `POST /orders`, `kafka.publish`, `kafka.consume`
- **Custom Attributes:** `order.id`, `customer.id`, `product.id`, `quantity`
- **Error Tracking:** Exceptions and error states are captured

---

## 🚨 **Troubleshooting**

### **Environment Issues**

**Problem:** Services can't connect to infrastructure
```bash
# Check if infrastructure is running
docker compose ps

# Should show postgres, kafka, zookeeper as "Up"
# If not, restart infrastructure
docker compose down && docker compose up -d
```

**Problem:** Port conflicts
```bash
# Check what's using the ports
lsof -i :5432  # PostgreSQL
lsof -i :9092  # Kafka  
lsof -i :8080  # Order Service

# Kill conflicting processes or change ports in docker-compose.yml
```

### **Configuration Issues**

**Problem:** Environment variables not loaded
```bash
# For Order Service (Java)
cd services/order-service
source .env
echo $DATABASE_URL  # Should show the database URL

# For Inventory Service (Go)
cd services/inventory-service  
source .env
echo $KAFKA_BROKER  # Should show kafka broker
```

**Problem:** Grafana Cloud connection fails
```bash
# Check your credentials are base64 encoded correctly
echo -n "instanceId:token" | base64

# Test the connection
curl -H "Authorization: Basic YOUR_TOKEN" \
  https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces
```

### **Service Issues**

**Problem:** Order Service won't start
```bash
# Check if OpenTelemetry agent exists
ls -la services/order-service/opentelemetry-javaagent.jar

# If missing, download it:
cd services/order-service
wget https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

**Problem:** Inventory Service won't start
```bash
# Check Go is installed
go version

# Check dependencies
cd services/inventory-service
go mod tidy
go run .
```

**Problem:** Kafka connection errors
```bash
# Test Kafka is accessible
docker exec distributed-tracing-otel-demo-kafka-1 kafka-topics.sh \
  --bootstrap-server localhost:9092 --list

# Should show topics, including 'order-events'
```

### **Tracing Issues**

**Problem:** No traces in Grafana Cloud
1. **Check credentials** - Verify your Grafana Cloud auth token
2. **Check endpoints** - Ensure OTLP endpoints are correct for your region
3. **Check network** - Services must reach `otlp-gateway-prod-eu-west-2.grafana.net`
4. **Check service logs** - Look for OTLP export errors

**Problem:** Traces are incomplete
1. **Check all services are instrumented** - Both Java agent and Go OTEL
2. **Check trace context propagation** - Kafka headers should contain trace info
3. **Check sampling** - Default is 100% sampling in development

### **Docker Issues**

**Problem:** Build failures
```bash
# Clean Docker cache
docker system prune -a --volumes

# Rebuild from scratch  
docker compose -f docker-compose.full.yml build --no-cache
```

**Problem:** Container networking issues
```bash
# Check internal Docker network
docker network ls
docker network inspect distributed-tracing-otel-demo_default

# Services should be on same network
```

---

## 🎯 **Next Steps**

### **Extend the Demo**
1. **Add more services** - Create additional microservices with tracing
2. **Add metrics** - Enable OpenTelemetry metrics collection
3. **Add alerting** - Set up Grafana alerts on trace data
4. **Add sampling** - Configure trace sampling for production loads

### **Production Considerations**
1. **Security** - Use proper authentication and TLS
2. **Performance** - Configure appropriate sampling rates  
3. **Monitoring** - Add health checks and monitoring
4. **Deployment** - Use Kubernetes or similar orchestration

### **Learning Resources**
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Grafana Cloud Documentation](https://grafana.com/docs/grafana-cloud/)
- [Distributed Tracing Best Practices](https://opentelemetry.io/docs/best-practices/)

---

## 📝 **Project Structure**

```
distributed-tracing-otel-demo/
├── docker-compose.yml              # Infrastructure only (default)
├── docker-compose.full.yml         # Full stack in Docker
├── README.md                       # This comprehensive guide
├── services/
│   ├── order-service/              # Java Spring Boot service
│   │   ├── .env                    # Local development config  
│   │   ├── .env.example            # Configuration template
│   │   ├── Dockerfile              # Docker build definition
│   │   ├── pom.xml                 # Maven dependencies
│   │   ├── opentelemetry-javaagent.jar  # OTEL Java agent
│   │   └── src/                    # Java source code
│   └── inventory-service/          # Go service
│       ├── .env                    # Local development config
│       ├── .env.example            # Configuration template  
│       ├── Dockerfile              # Docker build definition
│       ├── go.mod                  # Go dependencies
│       ├── main.go                 # Application entry point
│       └── internal/               # Go source code
└── .vscode/                        # VS Code debug configurations
    └── launch.json                 # Debug settings
```

---

**🎉 Congratulations!** You now have a complete distributed tracing setup. Every request flows through your system with full observability, giving you insights into performance, errors, and system behavior.

**Questions?** Check the troubleshooting section above or create an issue in the repository.
