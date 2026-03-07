#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOCAL_RUN_DIR="${ROOT_DIR}/local_run"

# Load environment variables from .env if it exists
if [ -f "${ROOT_DIR}/.env" ]; then
  echo "[dev-run] loading environment variables from .env..."
  export $(grep -v '^#' "${ROOT_DIR}/.env" | xargs)
fi

# Store PIDs for cleanup
PIDS=()

# Cleanup function
cleanup() {
  echo ""
  echo "[dev-run] shutting down all processes..."
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      echo "[dev-run] killing process $pid"
      kill "$pid" 2>/dev/null || true
    fi
  done
  # Wait a moment for graceful shutdown
  sleep 1
  # Force kill if still running
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      echo "[dev-run] force killing process $pid"
      kill -9 "$pid" 2>/dev/null || true
    fi
  done
  echo "[dev-run] cleanup completed"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

"${ROOT_DIR}/build.sh"

echo "[dev-run] preparing local_run directory..."

mkdir -p "${LOCAL_RUN_DIR}/config" "${LOCAL_RUN_DIR}/data"

cp "${ROOT_DIR}/dist/openfi-server" "${LOCAL_RUN_DIR}/openfi-server"

# Create config.json from sample if it doesn't exist
if [[ ! -f "${LOCAL_RUN_DIR}/config/config.json" ]]; then
  echo "[dev-run] creating config.json from sample..."
  cp "${ROOT_DIR}/config/config.sample.json" "${LOCAL_RUN_DIR}/config/config.json"
fi

# Start frontend-www dev server
echo "[dev-run] starting frontend-www dev server..."
cd "${ROOT_DIR}/frontend-www"
npm run dev &
PIDS+=($!)
echo "[dev-run] frontend-www started (PID: $!)"

# Start frontend-admin dev server
echo "[dev-run] starting frontend-admin dev server..."
cd "${ROOT_DIR}/frontend-admin"
npm run dev &
PIDS+=($!)
echo "[dev-run] frontend-admin started (PID: $!)"

# Wait a moment for frontend servers to initialize
sleep 2

# Start server
echo "[dev-run] starting server..."
cd "${LOCAL_RUN_DIR}"
OPENFI_WWW_DEV_SERVER="http://127.0.0.1:9334" \
OPENFI_ADMIN_DEV_SERVER="http://127.0.0.1:9335" \
./openfi-server -config ./config/config.json &
PIDS+=($!)
echo "[dev-run] server started (PID: $!)"

# Wait for all background processes
echo "[dev-run] all services started. Press Ctrl+C to stop."
wait
