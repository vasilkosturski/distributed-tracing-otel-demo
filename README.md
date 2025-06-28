# Distributed Tracing Demo with OpenTelemetry and Grafana Cloud

## 📝 Introduction

This project demonstrates end-to-end distributed tracing in a microservices architecture using OpenTelemetry and Grafana Cloud. You'll see how requests flow through a Java-based Order Service and a Go-based Inventory Service, with asynchronous communication via Kafka and data persistence in PostgreSQL. All observability data is collected and visualized in Grafana Cloud, providing deep insight into system behavior.

**What you'll achieve:**
- Understand distributed tracing across multiple services and technologies
- See how trace context is propagated through HTTP and Kafka
- Visualize complete request flows in Grafana Cloud
- Learn how to combine OpenTelemetry auto-instrumentation with manual instrumentation via the SDK for full control and visibility
- See how logs are correlated with trace IDs, enabling you to view all logs for a trace or a specific span in context

**System Workflow:**

![Workflow](docs/workflow-diagram.png)

*This diagram shows the end-to-end workflow of requests, events, and traces in the system.*

Below is a brief description of each step in the workflow:

1. The API client sends a request to the Order Service.  
2. The Order Service stores the order and publishes an event to Kafka.  
3. The Inventory Service receives the event and processes inventory.  
4. The Inventory Service publishes an inventory reserved event to Kafka.  
5. The Order Service updates the order status based on the inventory event.

Throughout this workflow, tracking data (spans) is sent to Grafana Cloud: the Java service sends both automatically captured spans and manual spans created through the SDK, while the Inventory Service sends only manual spans; all tracking data is linked via the trace id to provide a complete view of each request.

**Example distributed trace in Grafana Cloud:**  
*_(Add your screenshot here)_*
![Grafana Cloud Trace Example](docs/grafana-trace-example.png)

## 🚀 Quick Start

### **Step 1: Download OpenTelemetry Java Agent**

> **TODO:** Review and set the correct path for the OpenTelemetry Java agent JAR file download location below before publishing!

Download the OpenTelemetry Java agent to `services/order-service/` directory:

```bash
# Download the agent to the exact location
curl -L -o services/order-service/opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

**Note:** The JAR file must be placed in `services/order-service/` directory.

### **Step 2: Configure Grafana Cloud Credentials**

> **TODO:** Review and set proper Grafana Cloud URLs and instructions below before publishing!

For detailed setup instructions and advanced configuration options, see the [Grafana Cloud OTLP documentation](https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/#manual-opentelemetry-setup-for-advanced-users).

1. **Get your Grafana Cloud API key:**
   - Sign up at [grafana.com](https://grafana.com)
   - Create a new stack
   - Go to 'Access Policies' → 'API Keys'
   - Create a new API key

2. **Update environment files:**
   ```bash
   # Copy template files
   cp services/order-service/.env.example services/order-service/.env
   cp services/inventory-service/.env.example services/inventory-service/.env
   ```

3. **Add your credentials to both .env files:**
   - Replace `YOUR_ENCODED_CREDENTIALS_HERE` with your actual Grafana credentials
   - Format: `Basic <your-base64-encoded-credentials>`

### **Step 3: Run the Demo**

```bash
# Start all services
docker compose -f docker-compose.full.yml up -d --build
```

### **Step 4: Test It!**

Create a test request:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "product_id": "550e8400-e29b-41d4-a716-446655440001", 
    "quantity": 2
  }'
```

### **Step 5: View Traces in Grafana Cloud**

> **TODO:** Review and update the instructions below with the exact path or navigation steps for your Grafana Cloud instance, including any direct URLs if possible.

1. Go to your Grafana Cloud instance
2. Navigate to "Explore" → "Tempo"
3. Search for traces by service name: `order-service` or `inventory-service`
4. You should see the distributed trace flow!

## 🛠️ Technology Stack

- **Order Service**: Java 17, Spring Boot, OpenTelemetry Java Agent
- **Inventory Service**: Go 1.24, Kafka consumer/producer, OpenTelemetry Go SDK
- **Message Queue**: Apache Kafka with Zookeeper
- **Database**: PostgreSQL 13
- **Observability**: OpenTelemetry + Grafana Cloud
  - **Note**: This demo currently exports only traces and logs via OpenTelemetry. Metrics are not exported.
- **Containerization**: Docker & Docker Compose

## 🔍 **Services Overview**

### **Order Service (Java/Spring Boot)**
- **Purpose**: Handles HTTP requests and publishes events to Kafka
- **Instrumentation**: 
  - **Auto-instrumentation** via OpenTelemetry Java Agent (HTTP, Database, Kafka operations)
  - **Manual spans** for business logic and custom operations
- **Communication**: Publishes order events to Kafka topics

### **Inventory Service (Go)**
- **Purpose**: Consumes Kafka events and processes inventory updates
- **Instrumentation**: OpenTelemetry Go SDK with manual instrumentation
- **Communication**: Subscribes to Kafka topics for order events
- **Note**: This service does not expose an HTTP API; it is a pure Kafka consumer/producer.

## 🎯 **Instrumentation Strategy**

### **Auto-instrumentation (Java OTEL Agent)**
The OpenTelemetry Java Agent automatically traces:
- HTTP requests/responses (Spring Boot endpoints)
- Database operations (JDBC queries)
- Kafka producer/consumer operations
- HTTP client calls
- Framework-specific operations

### **Manual Instrumentation**
Custom spans for business logic:
- Order processing workflows
- Inventory validation logic
- Business event publishing
- Custom attributes and context

### **Combined Approach**
```java
// Auto: HTTP span created by Java agent
@PostMapping("/orders")
public ResponseEntity<?> createOrder(@RequestBody CreateOrderRequest request) {
    
    // Manual: Order creation span
    Span createOrderSpan = tracer.spanBuilder("create_order")
        .setAttribute("customer.id", request.getCustomerId())
        .startSpan();
    
    try (var scope = createOrderSpan.makeCurrent()) {
        // Auto: Database span created by Java agent
        Order order = orderService.createOrder(request);
        
        // Manual: Custom event publishing span
        Span eventSpan = tracer.spanBuilder("publish-order-event")
            .setAttribute("order.id", order.getId())
            .startSpan();
        
        try (var eventScope = eventSpan.makeCurrent()) {
            // Auto: Kafka producer span created by Java agent
            kafkaTemplate.send("order-created", event);
        } finally {
            eventSpan.end();
        }
        
        return ResponseEntity.ok(order);
    } finally {
        createOrderSpan.end();
    }
}
```

This combination provides complete visibility into both technical operations and business logic.

## 🧪 Testing

### **API Endpoints**

**Order Service** (http://localhost:8080):
- `GET /orders` - List all orders
- `POST /orders` - Create a new order

**Example order creation:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "product_id": "550e8400-e29b-41d4-a716-446655440001", 
    "quantity": 2
  }'
```

### **E2E Test**
This automated test script validates the complete end-to-end flow by creating an order and verifying that all services work together correctly, including the distributed tracing across the entire system.
```bash
./e2e-test.sh
```

## **Docker Compose Files**

The project provides two Docker Compose configurations for different use cases:

- `docker-compose.yml` - Basic services (PostgreSQL, Kafka, Zookeeper) for development and testing without the application services
- `docker-compose.full.yml` - Complete setup with all services including the Order Service and Inventory Service, ready for the full distributed tracing demo

## 📚 Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Grafana Cloud Documentation](https://grafana.com/docs/grafana-cloud/)
- [Spring Boot Documentation](https://spring.io/projects/spring-boot)
- [Go Documentation](https://golang.org/doc/)
- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)