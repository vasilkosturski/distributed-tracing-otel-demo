# Build stage
FROM golang:1.24-alpine AS build
WORKDIR /app

# Install git for Go modules
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary from the new entry point
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o inventory-service ./cmd/server

# Runtime stage
FROM alpine:latest
WORKDIR /app

# Install ca-certificates for HTTPS calls to Grafana Cloud
RUN apk --no-cache add ca-certificates

# Copy the binary
COPY --from=build /app/inventory-service .

# Run the service
ENTRYPOINT ["./inventory-service"]
