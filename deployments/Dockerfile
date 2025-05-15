# Build stage
FROM golang:1.24.3-alpine AS builder

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

# Build the application with explicit GOARCH and CGO_ENABLED=0 for better portability
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -o /app/bin/collector ./cmd/collector

# Final stage
FROM alpine@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

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
