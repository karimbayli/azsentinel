#!/usr/bin/env bash
# ============================================================
# Sentinel V2 — Central Server Deployment Script
# ============================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🚀 Deploying Sentinel V2 Central Server..."

cd "$PROJECT_DIR"

# Check for .env
if [ ! -f .env ]; then
    echo "❌ .env file not found. Copy .env.example to .env and configure it."
    exit 1
fi

# Pull latest images
echo "📦 Pulling latest images..."
docker compose -f deployments/docker-compose.central.yml pull

# Build central server
echo "🔨 Building central server..."
docker compose -f deployments/docker-compose.central.yml build central

# Start services
echo "▶️  Starting services..."
docker compose -f deployments/docker-compose.central.yml up -d

# Wait for health
echo "⏳ Waiting for services to become healthy..."
sleep 10

# Check health
if curl -sf http://localhost:8080/healthz > /dev/null 2>&1; then
    echo "✅ Central server is healthy!"
else
    echo "⚠️  Central server may still be starting. Check logs with:"
    echo "   docker compose -f deployments/docker-compose.central.yml logs -f central"
fi

echo ""
echo "📊 Grafana: http://localhost:3000"
echo "🔗 API:     http://localhost:8080/api/v1/status"
echo "📋 Health:  http://localhost:8080/healthz"
