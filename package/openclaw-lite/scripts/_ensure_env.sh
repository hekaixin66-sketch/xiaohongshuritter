#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  exit 0
fi

if grep -q '^XHS_DOCKERFILE=' "$ENV_FILE"; then
  exit 0
fi

ARCH="$(uname -m 2>/dev/null || echo unknown)"
DOCKERFILE="Dockerfile"

case "$ARCH" in
  arm64|aarch64)
    DOCKERFILE="Dockerfile.arm64"
    ;;
esac

{
  echo ""
  echo "# Dockerfile selected by install/update script"
  echo "XHS_DOCKERFILE=$DOCKERFILE"
} >> "$ENV_FILE"

echo "Selected Dockerfile for host architecture '$ARCH': $DOCKERFILE"
