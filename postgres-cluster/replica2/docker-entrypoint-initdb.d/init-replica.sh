#!/bin/bash
set -e

# Wait until primary is available
until pg_isready -h $PRIMARY_HOST -p 5432; do
  echo "Waiting for primary to be ready..."
  sleep 2
done

# Perform base backup
rm -rf /var/lib/postgresql/data/*
PGPASSWORD=$POSTGRES_PASSWORD pg_basebackup -h $PRIMARY_HOST -D /var/lib/postgresql/data -U $POSTGRES_USER -Fp -Xs -P -R

# Then the main postgres docker entrypoint will continue

