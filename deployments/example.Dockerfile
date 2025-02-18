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

# Build the example application
RUN go build -o /app/bin/example ./example

# Final stage
FROM alpine:latest

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
