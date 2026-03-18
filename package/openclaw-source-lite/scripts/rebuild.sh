#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_CMD="$($ROOT_DIR/scripts/_compose.sh)"

cd "$ROOT_DIR"
$COMPOSE_CMD down
docker image rm xiaohongshuritter-arm64:local >/dev/null 2>&1 || true
$COMPOSE_CMD build --no-cache
$COMPOSE_CMD up -d
$COMPOSE_CMD logs --tail=120
