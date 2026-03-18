#!/usr/bin/env bash
set -euo pipefail

if docker compose version >/dev/null 2>&1; then
  echo "docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  echo "docker-compose"
else
  echo "Docker Compose not found. Install Docker first." >&2
  exit 1
fi
