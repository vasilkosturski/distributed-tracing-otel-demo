# Stage 1: Build the Go binary (with Delve, ARM64-compatible)
FROM golang:1.24 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Install Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Build binary with debug info (no static build!)
# Avoid CGO_ENABLED=0 so Delve can use libc from Go image
RUN GOOS=linux GOARCH=arm64 go build -gcflags="all=-N -l" -o inventoryservice main.go

# Stage 2: Use full Go base image for matching GLIBC (no slim Debian)
FROM golang:1.24

WORKDIR /app

# Install debugging tools (optional)
RUN apt-get update && apt-get install -y \
    gdb \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary and delve
COPY --from=builder /app/inventoryservice .
COPY --from=builder /go/bin/dlv /usr/local/bin/dlv

EXPOSE 40000

#ENTRYPOINT ["dlv", "exec", "./inventoryservice", "--headless", "--listen=:40000", "--api-version=2", "--accept-multiclient"]
ENTRYPOINT ["./inventoryservice"]
