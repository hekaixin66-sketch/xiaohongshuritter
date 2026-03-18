#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PORT="${XHS_PORT:-18060}"
URL="http://127.0.0.1:${PORT}/health"

if command -v curl >/dev/null 2>&1; then
  curl -fsS "$URL" && echo
else
  wget -qO- "$URL" && echo
fi
