FROM golang:1.24.3-alpine

# Install goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Copy migrations
WORKDIR /migrations
COPY migrations/*.sql ./

# Create entrypoint script
COPY deployments/run-migrations.sh /run-migrations.sh
RUN chmod +x /run-migrations.sh

ENTRYPOINT ["/run-migrations.sh"]
