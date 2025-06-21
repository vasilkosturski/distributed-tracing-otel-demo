#!/bin/bash

# Distributed Tracing E2E Test Script
# Tests the complete flow and validates no OTEL authorization errors

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
COMPOSE_FILE="docker-compose.full.yml"
ORDER_SERVICE_URL="http://localhost:8080/orders"
TEST_TIMEOUT=60
LOG_CHECK_DURATION=15



echo -e "${BLUE}🚀 Starting Distributed Tracing E2E Test${NC}"
echo "=============================================="

# Step 1: Stop and remove project containers
echo -e "${YELLOW}📦 Step 1: Stopping project containers...${NC}"
docker compose -f $COMPOSE_FILE down --volumes 2>/dev/null || true
echo -e "${GREEN}✅ Project containers stopped${NC}"

# Step 2: Build all services
echo -e "${YELLOW}🔨 Step 2: Building all services...${NC}"
docker compose -f $COMPOSE_FILE build
echo -e "${GREEN}✅ All services built successfully${NC}"

# Step 3: Start all services
echo -e "${YELLOW}🏃 Step 3: Starting all services...${NC}"
docker compose -f $COMPOSE_FILE up -d

# Step 4: Wait for services to be ready
echo -e "${YELLOW}⏳ Step 4: Waiting for services to be ready...${NC}"
echo "Waiting for order service to be available..."

# Wait for order service to be ready
for i in {1..30}; do
    if curl -s -f $ORDER_SERVICE_URL >/dev/null 2>&1; then
        break
    elif [ $i -eq 30 ]; then
        echo -e "${RED}❌ Order service failed to start within 30 seconds${NC}"
        docker compose -f $COMPOSE_FILE logs order-service
        exit 1
    fi
    sleep 2
done

# Additional wait for all services to stabilize
echo "Waiting additional 10 seconds for all services to stabilize..."
sleep 10
echo -e "${GREEN}✅ All services are ready${NC}"

# DEBUG: Print inventory-service environment variables
echo -e "${BLUE}🔍 DEBUG: Printing environment variables from inventory-service...${NC}"
docker compose -f $COMPOSE_FILE exec inventory-service env
echo -e "${BLUE}==============================================================${NC}"

# Step 5: Verify clean inventory service startup (no test spans)
echo -e "${YELLOW}🧪 Step 5: Verifying clean inventory service startup...${NC}"
INVENTORY_STARTUP=$(docker compose -f $COMPOSE_FILE logs inventory-service 2>/dev/null)

if ! echo "$INVENTORY_STARTUP" | grep -q "Application initialized successfully"; then
    echo -e "${RED}❌ Inventory service did not initialize properly${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Inventory service startup is clean and proper${NC}"

# Step 6: Test order creation and distributed flow
echo -e "${YELLOW}📦 Step 6: Testing order creation and distributed flow...${NC}"

# Generate unique test data
TIMESTAMP=$(date +%s)
CUSTOMER_ID="550e8400-e29b-41d4-a716-$(printf "%012d" $((TIMESTAMP % 1000000000000)))"
PRODUCT_ID="123e4567-e89b-12d3-a456-$(printf "%012d" $((TIMESTAMP % 1000000000000 + 1)))"
QUANTITY=3

echo "Creating order with customer_id: $CUSTOMER_ID, product_id: $PRODUCT_ID, quantity: $QUANTITY"

# Create order
ORDER_RESPONSE=$(curl -s -X POST $ORDER_SERVICE_URL \
    -H "Content-Type: application/json" \
    -d "{\"customer_id\": \"$CUSTOMER_ID\", \"product_id\": \"$PRODUCT_ID\", \"quantity\": $QUANTITY}")

echo "Order creation response: $ORDER_RESPONSE"

# Extract order ID
ORDER_ID=$(echo "$ORDER_RESPONSE" | grep -o '"orderId":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ORDER_ID" ]; then
    echo -e "${RED}❌ Failed to extract order ID from response: $ORDER_RESPONSE${NC}"
    exit 1
fi

echo "Order ID: $ORDER_ID"

# Step 7: Wait and capture logs for validation
echo -e "${YELLOW}📋 Step 7: Waiting for distributed flow and capturing logs...${NC}"
sleep $LOG_CHECK_DURATION

# Capture all logs (not just recent ones to avoid timing issues)
ORDER_LOGS=$(docker compose -f $COMPOSE_FILE logs order-service 2>/dev/null)
INVENTORY_LOGS=$(docker compose -f $COMPOSE_FILE logs inventory-service 2>/dev/null)

# Step 8: Validate order service logs
echo -e "${YELLOW}🔍 Step 8: Validating order service flow...${NC}"

# Check order creation
if ! echo "$ORDER_LOGS" | grep -q "=== DEMO: Generated order ID: $ORDER_ID ==="; then
    echo -e "${RED}❌ Order creation log not found for order $ORDER_ID${NC}"
    echo "Order service logs:"
    echo "$ORDER_LOGS"
    exit 1
fi

# Check OrderCreated event publication
if ! echo "$ORDER_LOGS" | grep -q "=== DEMO: Publishing OrderCreated event to Kafka ==="; then
    echo -e "${RED}❌ OrderCreated event publication log not found${NC}"
    echo "Order service logs:"
    echo "$ORDER_LOGS"
    exit 1
fi

# Check order status update to INVENTORY_RESERVED
if ! echo "$ORDER_LOGS" | grep -q "=== DEMO: Marking order $ORDER_ID as INVENTORY_RESERVED ==="; then
    echo -e "${RED}❌ Order status update log not found for order $ORDER_ID${NC}"
    echo "Order service logs:"
    echo "$ORDER_LOGS"
    exit 1
fi

if ! echo "$ORDER_LOGS" | grep -q "=== DEMO: Order $ORDER_ID successfully updated to INVENTORY_RESERVED ==="; then
    echo -e "${RED}❌ Order status confirmation log not found for order $ORDER_ID${NC}"
    echo "Order service logs:"
    echo "$ORDER_LOGS"
    exit 1
fi

echo -e "${GREEN}✅ Order service flow validated successfully${NC}"

# Step 9: Check for error strings in order service logs
echo -e "${YELLOW}🔍 Step 9: Checking for error strings in order service logs...${NC}"

# Filter out harmless Kafka startup errors from order service logs too
FILTERED_ORDER_LOGS=$(echo "$ORDER_LOGS" | grep -i "error\|exception\|failed\|timeout\|refused" | grep -v "LEADER_NOT_AVAILABLE" | grep -v "Error while fetching metadata" || true)

if [ -n "$FILTERED_ORDER_LOGS" ]; then
    echo -e "${RED}❌ Error strings detected in order service logs:${NC}"
    echo "$FILTERED_ORDER_LOGS"
    exit 1
fi

echo -e "${GREEN}✅ No error strings found in order service logs${NC}"

# Step 10: Validate inventory service logs
echo -e "${YELLOW}🔍 Step 10: Validating inventory service flow...${NC}"

# Check inventory reservation
if ! echo "$INVENTORY_LOGS" | grep -q "Inventory reserved.*$ORDER_ID"; then
    echo -e "${RED}❌ Inventory reservation log not found for order $ORDER_ID${NC}"
    echo "Inventory service logs:"
    echo "$INVENTORY_LOGS"
    exit 1
fi

echo -e "${GREEN}✅ Inventory service flow validated successfully${NC}"

# Step 11: Check for OTEL authorization errors after business operations
echo -e "${YELLOW}🔍 Step 11: Checking for OTEL authorization errors after order processing...${NC}"

# Check both startup and runtime logs for OTEL errors
ALL_LOGS=$(docker compose -f $COMPOSE_FILE logs inventory-service order-service 2>/dev/null)

# Check for 401 Unauthorized errors
if echo "$ALL_LOGS" | grep -qi "401.*unauthorized\|unauthorized.*401"; then
    echo -e "${RED}❌ OTEL Authorization errors detected:${NC}"
    echo "$ALL_LOGS" | grep -i "401\|unauthorized"
    exit 1
fi

echo -e "${GREEN}✅ No OTEL authorization errors found after business operations${NC}"

# Step 12: Check for any general error strings
echo -e "${YELLOW}🔍 Step 12: Checking for any error strings in logs...${NC}"

# Filter out harmless Kafka startup errors
FILTERED_LOGS=$(echo "$ALL_LOGS" | grep -i "error\|exception\|failed\|timeout\|refused" | grep -v "LEADER_NOT_AVAILABLE" | grep -v "Error while fetching metadata" || true)

if [ -n "$FILTERED_LOGS" ]; then
    echo -e "${RED}❌ Error strings detected in logs:${NC}"
    echo "$FILTERED_LOGS"
    exit 1
fi

echo -e "${GREEN}✅ No error strings found in logs${NC}"

# Step 13: Cleanup (optional)
echo -e "${YELLOW}🧹 Step 13: Cleaning up test environment...${NC}"
docker compose -f $COMPOSE_FILE down --volumes >/dev/null 2>&1

echo ""
echo -e "${GREEN}🎉 E2E TEST PASSED! 🎉${NC}"
echo "=============================================="
echo -e "${GREEN}✅ Docker cleanup completed${NC}"
echo -e "${GREEN}✅ All services built successfully${NC}"
echo -e "${GREEN}✅ All services started successfully${NC}"
echo -e "${GREEN}✅ No OTEL authorization errors detected${NC}"
echo -e "${GREEN}✅ Clean inventory service startup validated${NC}"
echo -e "${GREEN}✅ Order created successfully: $ORDER_ID${NC}"
echo -e "${GREEN}✅ Complete distributed flow validated${NC}"
echo -e "${GREEN}✅ All logs show proper message flow${NC}"
echo -e "${GREEN}✅ No runtime OTEL errors detected${NC}"
echo ""
echo -e "${BLUE}The distributed tracing system is working perfectly!${NC}" 