#!/bin/bash

echo "🚀 Starting Distributed Tracing Demo in Debug Mode"
echo "=================================================="

# Change to project root
cd "$(dirname "$0")"

# Check if infrastructure is running
echo "📋 Checking infrastructure..."
if ! docker compose ps | grep -q "Up"; then
    echo "❌ Infrastructure not running. Starting..."
    docker compose up -d
    echo "⏳ Waiting for infrastructure to be ready..."
    sleep 10
fi

echo "✅ Infrastructure ready"
echo ""

# Start Java service
echo "🔥 Starting Java Order Service..."
cd services/order-service
source .env
mvn spring-boot:run -Dspring-boot.run.jvmArguments="-javaagent:./opentelemetry-javaagent.jar" &
JAVA_PID=$!
echo "📟 Java service PID: $JAVA_PID"

# Wait a bit for Java to start
sleep 8

# Start Go service  
echo ""
echo "🔥 Starting Go Inventory Service..."
cd ../inventory-service
source .env
go run . &
GO_PID=$!
echo "📟 Go service PID: $GO_PID"

echo ""
echo "🎉 Both services starting!"
echo "📊 Java Order Service: http://localhost:8080"
echo "📋 Database UI: http://localhost:8000"
echo ""
echo "🧪 Test with:"
echo "curl -X POST http://localhost:8080/orders -H 'Content-Type: application/json' -d '{\"customer_id\": \"123e4567-e89b-12d3-a456-426614174000\", \"product_id\": \"987fcdeb-51a2-43d1-9c47-123456789abc\", \"quantity\": 2}'"
echo ""
echo "🛑 To stop services:"
echo "kill $JAVA_PID $GO_PID"

# Keep script running and show process status
echo "⏳ Waiting for services to be ready..."
sleep 5

echo ""
echo "📈 Service Status:"
ps aux | grep -E "(spring-boot|go run)" | grep -v grep || echo "Services may still be starting..."

# Keep the script running
wait 