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

# Build the example application
RUN go build -o /app/bin/example ./example

# Final stage
FROM alpine@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

# Install necessary runtime packages
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/example /app/example

# Use non-root user
USER appuser

# Command to run (will be overridden by docker-compose)
ENTRYPOINT ["/app/example"]
