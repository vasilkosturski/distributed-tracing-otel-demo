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
![Grafana Cloud Trace Example](docs/grafana-trace-2.png)

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

For detailed setup instructions and advanced configuration options, see the [Grafana Cloud OTLP documentation](https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/#manual-opentelemetry-setup-for-advanced-users).

1. **Create account in Grafana Cloud**
   - Go to [grafana.com](https://grafana.com) and sign up for a free account
   - Complete the registration process

2. **Create a new Stack**
   ![Create Stack](docs/grafana-1.png)
   - After logging in, click "Create a stack" or "Add stack"
   - Choose a stack name (e.g., "distributed-tracing-demo")
   - Select your region and click "Create stack"

3. **Access OpenTelemetry Configuration**
   ![OpenTelemetry Configuration](docs/grafana-2.png)
   - In your stack dashboard, find the "OpenTelemetry" tile
   - Click on it to access the OpenTelemetry configuration

4. **Generate API Token**
   ![Generate API Token](docs/grafana-3.png)
   - Click "Generate a new API token" to create authentication credentials
   - This token will be used to authenticate your services with Grafana Cloud

5. **Create and Name the Token**
   ![Create Token](docs/grafana-4.png)
   - Give your token a descriptive name (e.g., "distributed-tracing-demo")
   - Click "Create token" to generate the credentials

6. **Save Your Credentials**
   ![Token Created](docs/grafana-5.png)
   - You'll see the created token and other environment variables
   - **Important**: Save these credentials securely - you'll need them to configure your project
   - The page will show the OTLP endpoint URL and authentication token

7. **Create Environment Files**
   
   **Important**: The `.env` files are not under source control for security reasons. You need to create them from the example files.
   
   ```bash
   # Copy example files to create your .env files
   cp services/order-service/.env.example services/order-service/.env
   cp services/inventory-service/.env.example services/inventory-service/.env
   ```

8. **Update Environment Variables**
   
   Use the credentials you saved from Grafana Cloud to populate both `.env` files. The example files contain realistic but invalid configurations that you need to replace with your actual credentials:
   
   **For Order Service** (`services/order-service/.env`):
   ```bash
   # Grafana Cloud OTLP endpoint (replace with your actual endpoint from step 6)
   OTEL_EXPORTER_OTLP_ENDPOINT=https://your-instance.grafana.net:443
   
   # Grafana Cloud authentication header (replace with your actual token from step 6)
   OTEL_EXPORTER_OTLP_HEADERS=authorization=Basic YOUR_ACTUAL_BASE64_ENCODED_CREDENTIALS
   
   # Service identification
   OTEL_SERVICE_NAME=order-service
   OTEL_RESOURCE_ATTRIBUTES=service.version=0.1.0,deployment.environment=demo
   ```
   
   **For Inventory Service** (`services/inventory-service/.env`):
   ```bash
   # Grafana Cloud OTLP endpoint (same as order-service)
   OTEL_ENDPOINT=https://your-instance.grafana.net:443
   
   # Grafana Cloud authentication header (same as order-service)
   OTEL_AUTH_HEADER=Basic YOUR_ACTUAL_BASE64_ENCODED_CREDENTIALS
   
   # Kafka configuration (already set in docker-compose)
   KAFKA_BROKER=kafka:9093
   ```

> **Note:**
> In this demo, both services export tracing and logging data *directly* to the Grafana Cloud OTLP endpoint. In a typical production setup, you would send telemetry to a local OpenTelemetry Collector, which then forwards data to Grafana Cloud. Direct export is used here for simplicity and ease of demonstration.

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

1. Go to your Grafana Cloud instance.
2. Navigate to the Traces section:
   
   ![Locate the Traces section in Grafana Cloud](docs/grafana-trace-1.png)

3. Click on a trace to view its details:
   
   ![View a single distributed trace](docs/grafana-trace-2.png)

4. Click on a specific span within the trace to see the correlated logs:
   
   ![See logs for a specific span in the trace](docs/grafana-trace-3.png)

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