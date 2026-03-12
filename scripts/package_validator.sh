#!/bin/bash
set -euo pipefail

# Build a validator distribution tarball:
# - bin/axond
# - scripts/init_mainnet.sh
# - scripts/mainnet_preflight.sh
# - install.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION="${VERSION:-v1.0.0}"
GOOS_TARGET="${GOOS_TARGET:-$(go env GOOS)}"
GOARCH_TARGET="${GOARCH_TARGET:-$(go env GOARCH)}"
OUT_DIR="$REPO_ROOT/dist"
SKIP_BUILD=false
BINARY_PATH=""

usage() {
  cat <<EOF
Usage:
  bash scripts/package_validator.sh [options]

Options:
  --version <v>     package version (default: ${VERSION})
  --os <goos>       target GOOS (default: ${GOOS_TARGET})
  --arch <goarch>   target GOARCH (default: ${GOARCH_TARGET})
  --out <dir>       output directory (default: ${OUT_DIR})
  --binary <path>   use prebuilt axond binary instead of building
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
    --binary) BINARY_PATH="$2"; shift 2 ;;
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

mkdir -p "$OUT_DIR"

DIST_NAME="axon-validator-${VERSION}-${GOOS_TARGET}-${GOARCH_TARGET}"
STAGE_DIR="$OUT_DIR/$DIST_NAME"
BIN_DIR="$STAGE_DIR/bin"
SCRIPTS_DIR="$STAGE_DIR/scripts"
TARBALL="$OUT_DIR/${DIST_NAME}.tar.gz"

rm -rf "$STAGE_DIR"
mkdir -p "$BIN_DIR" "$SCRIPTS_DIR"

COMMIT="$(git -C "$REPO_ROOT" log -1 --format='%H' 2>/dev/null || echo unknown)"
LDFLAGS="-X github.com/cosmos/cosmos-sdk/version.Name=axon \
-X github.com/cosmos/cosmos-sdk/version.AppName=axond \
-X github.com/cosmos/cosmos-sdk/version.Version=${VERSION} \
-X github.com/cosmos/cosmos-sdk/version.Commit=${COMMIT}"

if [[ -n "$BINARY_PATH" ]]; then
  if [[ ! -x "$BINARY_PATH" ]]; then
    echo "Provided binary is not executable: $BINARY_PATH"
    exit 1
  fi
  cp "$BINARY_PATH" "$BIN_DIR/axond"
elif [[ "$SKIP_BUILD" != true ]]; then
  echo "Building axond for ${GOOS_TARGET}/${GOARCH_TARGET}..."
  (
    cd "$REPO_ROOT"
    CGO_ENABLED=0 GOOS="$GOOS_TARGET" GOARCH="$GOARCH_TARGET" \
      go build -mod=readonly -ldflags "$LDFLAGS" -o "$BIN_DIR/axond" ./cmd/axond
  )
fi

if [[ ! -x "$BIN_DIR/axond" ]]; then
  echo "axond binary not found in $BIN_DIR"
  echo "Hint: run without --skip-build, or place binary at $BIN_DIR/axond"
  exit 1
fi

cp "$REPO_ROOT/scripts/init_mainnet.sh" "$SCRIPTS_DIR/"
cp "$REPO_ROOT/scripts/mainnet_preflight.sh" "$SCRIPTS_DIR/"

cat > "$STAGE_DIR/install.sh" <<'EOF'
#!/bin/bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
TARGET_BIN_DIR="$PREFIX/bin"

if [[ ! -x "bin/axond" ]]; then
  echo "bin/axond not found. Please run install.sh in extracted package root."
  exit 1
fi

mkdir -p "$TARGET_BIN_DIR"
install -m 0755 "bin/axond" "$TARGET_BIN_DIR/axond"

echo "Installed: $TARGET_BIN_DIR/axond"
echo "Version:"
"$TARGET_BIN_DIR/axond" version || true
echo
echo "Next:"
echo "  1) bash scripts/init_mainnet.sh --home ~/.axon-mainnet"
echo "  2) bash scripts/mainnet_preflight.sh --home ~/.axon-mainnet --binary $TARGET_BIN_DIR/axond"
EOF

cat > "$STAGE_DIR/README.txt" <<EOF
Axon Validator Package
======================

Package: ${DIST_NAME}

Contents:
  - bin/axond
  - scripts/init_mainnet.sh
  - scripts/mainnet_preflight.sh
  - install.sh

Install:
  bash install.sh

Or install to custom prefix:
  PREFIX=\$HOME/.local bash install.sh
EOF

chmod +x "$STAGE_DIR/install.sh" "$SCRIPTS_DIR/init_mainnet.sh" "$SCRIPTS_DIR/mainnet_preflight.sh"

(
  cd "$OUT_DIR"
  tar -czf "$TARBALL" "$DIST_NAME"
)

echo "Validator package created:"
echo "  $TARBALL"
