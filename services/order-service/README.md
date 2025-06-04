# Order Service

Java Spring Boot service that handles order creation and publishes events to Kafka with OpenTelemetry tracing.

## Configuration

All configuration is done via environment variables. Copy `.env.example` to `.env` and update with your values:

```bash
cp .env.example .env
```

### Environment Variables

- **Database**: `DATABASE_URL`, `DATABASE_USER`, `DATABASE_PASSWORD`
- **Kafka**: `KAFKA_BROKER`
- **OpenTelemetry**: All `OTEL_*` variables for tracing and logging

## Running Locally

### Option 1: IntelliJ IDEA
1. Import the project
2. Use the pre-configured "OrderServiceApplication" run configuration
3. It automatically loads the `.env` file and sets up the OpenTelemetry Java agent

### Option 2: VS Code
1. Open the workspace root
2. Use the "Launch Order Service (Java)" debug configuration
3. It loads environment variables from `.env` file

### Option 3: Command Line
```bash
# Make sure you have the OpenTelemetry Java agent
# Set environment variables from .env file
source .env
mvn spring-boot:run -Dspring-boot.run.jvmArguments="-javaagent:./opentelemetry-javaagent.jar"
```

### Option 4: Docker
```bash
# From project root
docker-compose up --build order-service
```

## API Endpoints

- `POST /api/orders` - Create a new order
- `GET /api/orders/{id}` - Get order by ID

## Dependencies

- PostgreSQL database
- Kafka broker
- OpenTelemetry Java agent (`opentelemetry-javaagent.jar`)

## Tracing

The service automatically sends traces to Grafana Cloud when properly configured. All HTTP requests, database queries, and Kafka messages are traced. 