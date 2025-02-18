# Build stage
FROM golang:1.24-alpine AS builder

# Install required system packages
RUN apk add --no-cache git make protoc protobuf-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
RUN go build -o /app/bin/collector ./cmd/collector

# Final stage
FROM alpine:latest

# Install necessary runtime packages
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/collector /app/collector

# Use non-root user
USER appuser

# Set default environment variables
ENV AMMO_COLLECTOR_SERVICE_NAME=ammo-collector \
    AMMO_COLLECTOR_LOG_LEVEL=INFO \
    AMMO_COLLECTOR_ENV_TYPE=PROD \
    AMMO_COLLECTOR_SERVER_HOST=0.0.0.0 \
    AMMO_COLLECTOR_GRPC_PORT=8090 \
    AMMO_COLLECTOR_HTTP_PORT=8091 \
    AMMO_COLLECTOR_METRICS_PROMETHEUS_ENDPOINT=0.0.0.0:8092

# Command to run
ENTRYPOINT ["/app/collector"]
