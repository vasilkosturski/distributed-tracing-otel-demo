# services/otel-go-example/Dockerfile

# Step 1: Build the application
FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /otel-go-example-app .

# Step 2: Create a production image using golang:1.24-alpine as runtime
FROM golang:1.24-alpine AS runtime 
# Using 'AS runtime' for clarity, though not strictly necessary if not multi-stage from here
WORKDIR /app

# Copy the executable from the builder stage
COPY --from=builder /otel-go-example-app /app/otel-go-example-app

# Ensure the binary is executable (alpine base images usually handle this, but good practice)
RUN chmod +x /app/otel-go-example-app

EXPOSE 8080
CMD ["/app/otel-go-example-app"]