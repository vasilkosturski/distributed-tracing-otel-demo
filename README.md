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

Download the OpenTelemetry Java agent to `services/order-service/` directory:

```bash
# Download the agent to the exact location
curl -L -o services/order-service/opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

**Note:** The JAR file must be placed in `services/order-service/` directory.

### **Step 2: Configure Grafana Cloud Credentials**

The following steps will guide you through generating and configuring your Grafana credentials for this project. For further details, see the [Grafana Cloud OTLP setup guide](https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/#manual-opentelemetry-setup-for-advanced-users).

1. **Create account in Grafana Cloud**
   - Go to [grafana.com](https://grafana.com) and sign up for a free account
   - Complete the registration process

2. **Create a new Stack**
   ![Create Stack](docs/grafana-1.png)
   - Choose an instance name
   - Select your region and click "Add stack"

3. **Access OpenTelemetry Configuration**
   ![OpenTelemetry Configuration](docs/grafana-2.png)
   - In your stack dashboard, find the "OpenTelemetry" tile
   - Click on it to access the OpenTelemetry configuration

4. **Generate API Token**
   ![Generate API Token](docs/grafana-3.png)
   - Click "Generate now" to create authentication credentials
   - This token will be used to authenticate your services with Grafana Cloud

5. **Create and Name the Token**
   ![Create Token](docs/grafana-4.png)
   - Give your token a name
   - Click "Create token" to generate the credentials

6. **Save Your Credentials**
   ![Token Created](docs/grafana-5.png)
   - You'll see the created token and other environment variables
   - **Important**: Save these credentials securely - you'll need them to configure your project
   - The page will show the OTLP endpoint URL and authentication token

7. **Create Environment Files**
   
   **Important**: The `.env` files are not under source control for security reasons. You need to create them from the `.env.example` files for both projects.
   
   ```bash
   # Copy example files to create your .env files
   cp services/order-service/.env.example services/order-service/.env
   cp services/inventory-service/.env.example services/inventory-service/.env
   ```

8. **Update Environment Variables**
   
   Update the following variables in each `.env` file with your Grafana Cloud credentials and endpoints:
   
   **For Order Service** (`services/order-service/.env`):
   ```env
   # Replace with your actual traces endpoint from Grafana Cloud
   OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://<your-grafana-instance>.grafana.net/otlp/v1/traces

   # Replace with your actual logs endpoint from Grafana Cloud
   OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=https://<your-grafana-instance>.grafana.net/otlp/v1/logs

   # Replace with your actual authentication header from Grafana Cloud
   OTEL_EXPORTER_OTLP_HEADERS="Authorization=Basic YOUR_BASE64_ENCODED_CREDENTIALS"
   ```
   
   **For Inventory Service** (`services/inventory-service/.env`):
   ```env
   # Replace with your actual authentication header from Grafana Cloud
   OTEL_AUTH_HEADER=Basic YOUR_BASE64_ENCODED_CREDENTIALS

   # Replace with your actual endpoint from Grafana Cloud (used for both traces and logs)
   OTEL_ENDPOINT=<your-grafana-instance>.grafana.net
   ```
   
   > **Note:**
   > The order service requires separate endpoints for traces and logs, while the inventory service uses a single endpoint for both. Only the endpoints and authentication header need to be modified for Grafana Cloud integration.

### **Step 3: Run the Demo**

```bash
# Start all services
docker compose -f docker-compose.full.yml up -d --build
```

### **Step 4: Test It!**

Create a test request by posting an order:

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

3. Click on the latest trace to view its details:
   
   ![View a single distributed trace](docs/grafana-trace-2.png)

4. You can also see the correlated logs for the span:
   
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
- **Purpose:**  
  - Handles HTTP requests to create new orders.
  - Publishes `OrderCreated` events to Kafka after saving orders to the database.
  - Consumes `InventoryReserved` events from Kafka and updates the corresponding order's status in the database.
- **Instrumentation:**  
  - **Auto-instrumentation** via OpenTelemetry Java Agent (HTTP, Database, Kafka operations)
  - **Manual spans** for business logic and custom operations
- **Communication:**  
  - Publishes order events to Kafka topics
  - Listens for inventory events from Kafka and updates orders accordingly

### **Inventory Service (Go)**
- **Purpose:**  
  - Consumes `OrderCreated` events from Kafka to process inventory reservations.
  - Publishes `InventoryReserved` events to Kafka after successfully reserving inventory.
- **Instrumentation:**  
  - Uses the OpenTelemetry Go SDK with manual instrumentation for tracing and logging.
- **Communication:**  
  - Listens for order events from Kafka and processes inventory updates.
  - Publishes inventory events to Kafka topics.

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

### **Example: Combining Auto-Instrumentation and Manual Instrumentation**

OpenTelemetry's Java agent automatically instruments common frameworks (HTTP, JDBC, Kafka, etc.), creating spans for technical operations. You can also use the OpenTelemetry SDK in your code to create custom spans for business logic. Both approaches use the same tracer and context, so all spans are linked into a single distributed trace.

**How it works in this project:**
- The Java agent (enabled via `-javaagent:opentelemetry-javaagent.jar`) auto-instruments HTTP endpoints, database queries, and Kafka operations in the Order Service.
- Manual instrumentation is used in business logic to create custom spans, add attributes, and record exceptions.

**Example from the project (OrderService.java):**

```java
// This method is called within an auto-instrumented HTTP span created by the Java agent for the /orders endpoint

// Manual span for business logic (child of the auto-instrumented HTTP span)
Span span = tracer.spanBuilder("create_order").startSpan();
try (Scope ignored = span.makeCurrent()) {
    // Custom attributes for business logic
    span.setAttribute("order.id", orderId.toString());
    // Auto-instrumented: JDBC span for the following query
    jdbcTemplate.update("INSERT INTO orders ...");
    // Auto-instrumented: Kafka span for publishing event
    kafkaTemplate.send("OrderCreated", event).get();
} catch (Exception e) {
    span.setStatus(StatusCode.ERROR, "Order creation failed");
    span.recordException(e);
    throw e;
} finally {
    span.end();
}
```

- The `tracer` used here is the same one managed by the OpenTelemetry SDK and recognized by the agent.
- All spans (auto and manual) are part of the same trace, and context is propagated automatically.

**SDK and Agent Integration:**
- The agent injects itself into the JVM and instruments supported libraries.
- The SDK (configured in your code) allows you to create custom spans and enrich traces with business-specific data.
- Context propagation ensures that all spans—whether created by the agent or manually—are linked together.