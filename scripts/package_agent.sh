#!/bin/bash
set -euo pipefail

# Build an agent-daemon distribution tarball:
# - bin/agent-daemon
# - install.sh
# - README.txt

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENT_DIR="$REPO_ROOT/tools/agent-daemon"

VERSION="${VERSION:-v1.0.0}"
GOOS_TARGET="${GOOS_TARGET:-$(go env GOOS)}"
GOARCH_TARGET="${GOARCH_TARGET:-$(go env GOARCH)}"
OUT_DIR="$REPO_ROOT/dist"
SKIP_BUILD=false

usage() {
  cat <<EOF
Usage:
  bash scripts/package_agent.sh [options]

Options:
  --version <v>     package version (default: ${VERSION})
  --os <goos>       target GOOS (default: ${GOOS_TARGET})
  --arch <goarch>   target GOARCH (default: ${GOARCH_TARGET})
  --out <dir>       output directory (default: ${OUT_DIR})
  --skip-build      skip binary build step
  --help            show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --os) GOOS_TARGET="$2"; shift 2 ;;
    --arch) GOARCH_TARGET="$2"; shift 2 ;;
    --out) OUT_DIR="$2"; shift 2 ;;
    --skip-build) SKIP_BUILD=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 1
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "go is required"
  exit 1
fi

if [[ ! -f "$AGENT_DIR/main.go" ]]; then
  echo "agent-daemon source not found: $AGENT_DIR/main.go"
  exit 1
fi

mkdir -p "$OUT_DIR"

DIST_NAME="axon-agent-daemon-${VERSION}-${GOOS_TARGET}-${GOARCH_TARGET}"
STAGE_DIR="$OUT_DIR/$DIST_NAME"
BIN_DIR="$STAGE_DIR/bin"
TARBALL="$OUT_DIR/${DIST_NAME}.tar.gz"

rm -rf "$STAGE_DIR"
mkdir -p "$BIN_DIR"

if [[ "$SKIP_BUILD" != true ]]; then
  echo "Building agent-daemon for ${GOOS_TARGET}/${GOARCH_TARGET}..."
  (
    cd "$AGENT_DIR"
    CGO_ENABLED=0 GOOS="$GOOS_TARGET" GOARCH="$GOARCH_TARGET" \
      go build -trimpath -ldflags "-s -w" -o "$BIN_DIR/agent-daemon" .
  )
fi

if [[ ! -x "$BIN_DIR/agent-daemon" ]]; then
  echo "agent-daemon binary not found in $BIN_DIR"
  echo "Hint: run without --skip-build, or place binary at $BIN_DIR/agent-daemon"
  exit 1
fi

cat > "$STAGE_DIR/install.sh" <<'EOF'
#!/bin/bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
TARGET_BIN_DIR="$PREFIX/bin"

if [[ ! -x "bin/agent-daemon" ]]; then
  echo "bin/agent-daemon not found. Please run install.sh in extracted package root."
  exit 1
fi

mkdir -p "$TARGET_BIN_DIR"
install -m 0755 "bin/agent-daemon" "$TARGET_BIN_DIR/agent-daemon"

echo "Installed: $TARGET_BIN_DIR/agent-daemon"
echo
echo "Example:"
echo "  $TARGET_BIN_DIR/agent-daemon --rpc http://72.62.251.50:8545 --private-key-file /path/to/key.txt --heartbeat-interval 100"
EOF

cp "$AGENT_DIR/README.md" "$STAGE_DIR/README.md"

cat > "$STAGE_DIR/README.txt" <<EOF
Axon Agent Daemon Package
=========================

Package: ${DIST_NAME}

Contents:
  - bin/agent-daemon
  - install.sh
  - README.md

Install:
  bash install.sh

Or install to custom prefix:
  PREFIX=\$HOME/.local bash install.sh
EOF

chmod +x "$STAGE_DIR/install.sh"

(
  cd "$OUT_DIR"
  tar -czf "$TARBALL" "$DIST_NAME"
)

echo "Agent package created:"
echo "  $TARBALL"
