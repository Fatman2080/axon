#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION="${VERSION:-v1.0.0}"
GOOS_TARGET="${GOOS_TARGET:-$(go env GOOS)}"
GOARCH_TARGET="${GOARCH_TARGET:-$(go env GOARCH)}"
OUT_DIR="$REPO_ROOT/dist"
VALIDATOR_BINARY="${VALIDATOR_BINARY:-}"

usage() {
  cat <<EOF
Usage:
  bash scripts/package_all.sh [options]

Options:
  --version <v>     package version (default: ${VERSION})
  --os <goos>       target GOOS (default: ${GOOS_TARGET})
  --arch <goarch>   target GOARCH (default: ${GOARCH_TARGET})
  --out <dir>       output directory (default: ${OUT_DIR})
  --validator-binary <path>  package prebuilt axond binary
  --help            show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --os) GOOS_TARGET="$2"; shift 2 ;;
    --arch) GOARCH_TARGET="$2"; shift 2 ;;
    --out) OUT_DIR="$2"; shift 2 ;;
    --validator-binary) VALIDATOR_BINARY="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 1
      ;;
  esac
done

echo "Packaging validator..."
if [[ -n "$VALIDATOR_BINARY" ]]; then
  bash "$SCRIPT_DIR/package_validator.sh" \
    --version "$VERSION" \
    --os "$GOOS_TARGET" \
    --arch "$GOARCH_TARGET" \
    --out "$OUT_DIR" \
    --binary "$VALIDATOR_BINARY"
else
  bash "$SCRIPT_DIR/package_validator.sh" \
    --version "$VERSION" \
    --os "$GOOS_TARGET" \
    --arch "$GOARCH_TARGET" \
    --out "$OUT_DIR"
fi

echo "Packaging agent-daemon..."
bash "$SCRIPT_DIR/package_agent.sh" \
  --version "$VERSION" \
  --os "$GOOS_TARGET" \
  --arch "$GOARCH_TARGET" \
  --out "$OUT_DIR"

echo "All packages created in: $OUT_DIR"
