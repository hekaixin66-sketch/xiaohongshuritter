#!/usr/bin/env bash
set -euo pipefail

PORT="${XHS_PORT:-18060}"
BASE_URL="http://127.0.0.1:${PORT}"

echo "[1/3] health check..."
curl -fsS "${BASE_URL}/health" >/dev/null
echo "ok"

echo "[2/3] accounts check..."
ACCOUNTS_JSON="$(curl -fsS "${BASE_URL}/api/v1/accounts")"
echo "$ACCOUNTS_JSON" | grep -q "\"success\":true" || {
  echo "accounts endpoint failed"
  exit 1
}
echo "ok"

echo "[3/3] login status check (default scope)..."
curl -fsS "${BASE_URL}/api/v1/login/status" >/dev/null
echo "ok"

echo "smoke test passed"
