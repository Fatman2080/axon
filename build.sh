#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="${ROOT_DIR}/dist"

mkdir -p "${DIST_DIR}"

echo "[build] building server binary..."
go build -o "${DIST_DIR}/openfi-server" ./src
echo "[build] output: ${DIST_DIR}/openfi-server"
