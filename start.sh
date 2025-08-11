#!/usr/bin/env bash
set -e  # Exit immediately if any command fails

DATA_DIR="./data"

echo "=== Creating local directory for the database ==="
mkdir -p "$DATA_DIR"

echo "=== Building Docker images ==="
docker compose build

echo "=== Starting containers ==="
docker compose up -d

echo
echo "=== All services are now running ==="
echo "Frontend (UI):  http://localhost:8088"
echo "Backend (API):  http://localhost:8080/api"
echo "Database file:  ${DATA_DIR}/storage.db"

