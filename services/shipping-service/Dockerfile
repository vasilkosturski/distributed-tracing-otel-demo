# Stage 1: Build the Go binary (static, ARM64-compatible)
FROM golang:1.24 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static build for Linux ARM64 (Apple Silicon)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -gcflags="all=-N -l" -a -installsuffix cgo -o shippingservice main.go

# Stage 2: Minimal runtime container
FROM debian:bullseye-slim

# Install runtime tools (if needed for debugging)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gdb \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the statically built binary from builder stage
COPY --from=builder /app/shippingservice .

# Optional: expose debugging port (e.g. for Delve)
EXPOSE 40000

ENTRYPOINT ["./shippingservice"]
