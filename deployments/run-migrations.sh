#!/bin/sh

# Wait for postgres to be ready
until goose -dir . postgres "$AMMO_COLLECTOR_DATABASE_URL" status > /dev/null 2>&1; do
  echo "Waiting for postgres..."
  sleep 1
done

# Run migrations
echo "Running migrations..."
goose -dir . postgres "$AMMO_COLLECTOR_DATABASE_URL" up

# Check migration status
goose -dir . postgres "$AMMO_COLLECTOR_DATABASE_URL" status
