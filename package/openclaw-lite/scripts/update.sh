#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$ROOT_DIR/../.." && pwd)"
COMPOSE_CMD="$("$ROOT_DIR/scripts/_compose.sh")"

DO_PULL=1
NO_CACHE=0
SHOW_LOGS=1

for arg in "$@"; do
  case "$arg" in
    --no-pull)
      DO_PULL=0
      ;;
    --no-cache|--full)
      NO_CACHE=1
      ;;
    --no-logs)
      SHOW_LOGS=0
      ;;
    *)
      echo "Unknown option: $arg" >&2
      echo "Usage: ./scripts/update.sh [--no-pull] [--no-cache|--full] [--no-logs]" >&2
      exit 1
      ;;
  esac
done

cd "$REPO_ROOT"

if [[ $DO_PULL -eq 1 ]]; then
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    CURRENT_BRANCH="$(git branch --show-current)"
    echo "Updating repository branch: ${CURRENT_BRANCH:-unknown}"
    git pull --ff-only
  else
    echo "Repository metadata not found, skipping git pull."
  fi
fi

cd "$ROOT_DIR"

"$ROOT_DIR/scripts/_ensure_env.sh"

BUILD_ARGS=()
if [[ $NO_CACHE -eq 1 ]]; then
  BUILD_ARGS+=(--no-cache)
fi

echo "Using compose command: $COMPOSE_CMD"
$COMPOSE_CMD build "${BUILD_ARGS[@]}"
$COMPOSE_CMD up -d
$COMPOSE_CMD ps

if [[ $SHOW_LOGS -eq 1 ]]; then
  $COMPOSE_CMD logs --tail=80
fi

echo "Update complete."
echo "Health check: http://127.0.0.1:${XHS_PORT:-18060}/health"
echo "MCP endpoint: http://127.0.0.1:${XHS_PORT:-18060}/mcp"
