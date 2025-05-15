# services/otel-go-example/Dockerfile

# Step 1: Build the application
# Ensure this Go version is >= the version in your go.mod (e.g., go 1.24.2)
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker layer caching for dependencies
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Verify dependencies
RUN go mod verify

# Copy the entire source code for this service
COPY . .

# Build the Go app, statically linked for Alpine
# Output executable named 'otel-go-example-app'
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /otel-go-example-app .

# Step 2: Create a production image using a Go-friendly Alpine base
FROM golang:1.24-alpine AS runtime
# This base image is chosen because:
# 1. It's Alpine, so it's small.
# 2. It's golang:alpine, so it has a Go-conducive environment which might be slightly
#    more robust for Go binaries than plain alpine:latest, especially regarding
#    expectations around libc (even for static binaries, sometimes very subtle things matter).
# 3. It matches the builder's Go version base, which is good practice.

WORKDIR /app

# Copy the executable from the builder stage
COPY --from=builder /otel-go-example-app /app/otel-go-example-app

# Ensure the binary is executable
RUN chmod +x /app/otel-go-example-app

# Standard port for the application
EXPOSE 8080

# Use sh -c "exec ..." to ensure the Go application becomes PID 1
# and its stdout/stderr are properly handled, especially with volumes.
# The optional "echo" and "sleep 1" can be removed if not needed for startup diagnostics,
# but the "exec" form is generally recommended.
CMD ["sh", "-c", "echo 'INFO: Starting otel-go-example-app...' && sleep 1 && exec /app/otel-go-example-app"]