# Feature Branch: Environment-Based Configuration

## 🎯 **Overview**
This branch transforms the distributed tracing demo from hardcoded configuration to a flexible, environment-based setup with two operational modes.

## 🚀 **Key Changes**

### **1. Environment-Based Configuration**
- **Java Service**: All config via environment variables (database, Kafka, OpenTelemetry)
- **Go Service**: Simplified config loading from environment only
- **Security**: Real credentials in `.env`, placeholders in `.env.example`

### **2. Dual Operational Modes**

#### **Mode 1: Hybrid Development (Default)**
```bash
docker compose up -d          # Infrastructure only
# Run services locally in IDE or terminal
```
- **Services**: Run locally (debuggable)
- **Infrastructure**: Docker (PostgreSQL, Kafka, Zookeeper)
- **Perfect for**: Daily development, debugging, testing

#### **Mode 2: Full Docker**
```bash
docker compose -f docker-compose.full.yml up --build
```
- **Everything**: Runs in Docker containers
- **Perfect for**: Integration testing, deployment simulation

### **3. Developer Experience Improvements**

#### **VS Code Integration**
- **Launch configurations** for both services
- **Environment loading** from `.env` files
- **Debugging support** with breakpoints

#### **IntelliJ Integration**
- **Pre-configured run configuration** with OpenTelemetry agent
- **Environment variables** loaded automatically

#### **Scripts & Documentation**
- **`start-debug.sh`**: One-command startup for both services
- **Service READMEs**: Detailed setup instructions
- **Updated main README**: Clear operational mode documentation

## 📁 **New Files**

```
├── docker-compose.yml           # Infrastructure only (default)
├── docker-compose.full.yml      # Full stack in Docker
├── start-debug.sh               # Dual service startup script
├── services/order-service/
│   ├── .env                     # Local development config
│   ├── .env.example             # Template with placeholders
│   ├── Dockerfile               # Multi-stage build with OpenTelemetry
│   └── README.md                # Service-specific documentation
└── .vscode/launch.json          # Debug configurations

```

## 🔧 **Modified Files**

- **`services/order-service/src/main/resources/application.yml`**: Removed hardcoded OpenTelemetry config (now using Java agent)
- **`services/inventory-service/.env`**: Updated with consistent format
- **`README.md`**: Complete rewrite with dual-mode documentation

## 🧪 **Testing Workflow**

### **Test Hybrid Mode**
```bash
# 1. Start infrastructure
docker compose up -d

# 2. Use VS Code debug configurations OR
cd services/order-service && source .env && mvn spring-boot:run -Dspring-boot.run.jvmArguments="-javaagent:./opentelemetry-javaagent.jar"
cd services/inventory-service && source .env && go run .

# 3. Test end-to-end
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "123e4567-e89b-12d3-a456-426614174000", "product_id": "987fcdeb-51a2-43d1-9c47-123456789abc", "quantity": 2}'
```

### **Test Full Docker Mode**
```bash
docker compose -f docker-compose.full.yml up --build
# Same curl test as above
```

## 🎊 **Benefits**

1. **Flexible Development**: Choose between local debugging or full Docker
2. **Consistent Configuration**: Same environment variables across all deployment modes
3. **Easy Onboarding**: Copy `.env.example` → `.env` and you're ready
4. **Production Ready**: Environment-based config works in any deployment environment
5. **Better DX**: IDE integration with proper debug support

## 🔀 **Merge Strategy**

1. **Test this branch** thoroughly in both modes
2. **Verify end-to-end tracing** works in both configurations  
3. **Create PR** from this branch to main
4. **Review and merge** when satisfied

This branch maintains backward compatibility while adding significant flexibility and developer experience improvements. 