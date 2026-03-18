#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p data images config

if [[ ! -f .env ]]; then
  cp .env.example .env
fi

if [[ ! -f config/accounts.json ]]; then
  cp config/accounts.json.example config/accounts.json
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
else
  echo "Docker Compose not found. Install Docker first."
  exit 1
fi

echo "Using compose command: $COMPOSE_CMD"
$COMPOSE_CMD build
$COMPOSE_CMD up -d
$COMPOSE_CMD ps

echo "Service started from repository source."
echo "Health check: http://127.0.0.1:${XHS_PORT:-18060}/health"
echo "MCP endpoint: http://127.0.0.1:${XHS_PORT:-18060}/mcp"
