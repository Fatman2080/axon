#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
RELEASE_NAME="openfi-${VERSION}"
RELEASE_DIR="${DIST_DIR}/${RELEASE_NAME}"
RELEASE_PACKAGE="${DIST_DIR}/${RELEASE_NAME}.tar.gz"

echo "[release] starting release build..."
echo "[release] version: ${VERSION}"

# Clean and prepare directories
echo "[release] preparing directories..."
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}/config"
mkdir -p "${RELEASE_DIR}/data"
mkdir -p "${RELEASE_DIR}/logs"
mkdir -p "${RELEASE_DIR}/assets/www"
mkdir -p "${RELEASE_DIR}/assets/admin"

# Build frontend-www
echo "[release] building frontend-www..."
cd "${ROOT_DIR}/frontend-www"
npm install
npm run build
cp -r dist/. "${RELEASE_DIR}/assets/www/"

# Build frontend-admin
echo "[release] building frontend-admin..."
cd "${ROOT_DIR}/frontend-admin"
npm install
npm run build
cp -r dist/. "${RELEASE_DIR}/assets/admin/"

# Build server
echo "[release] building server..."
cd "${ROOT_DIR}"
go build -o "${RELEASE_DIR}/openfi-server" ./src

# Generate release config
# If config.prod.json exists use it, otherwise generate from sample with release paths
echo "[release] generating config..."
if [[ -f "${ROOT_DIR}/config/config.prod.json" ]]; then
  cp "${ROOT_DIR}/config/config.prod.json" "${RELEASE_DIR}/config/config.json"
else
  cat > "${RELEASE_DIR}/config/config.json" << 'CONF'
{
  "appBaseUrl": "http://localhost:9333",
  "server": {
    "port": 9333,
    "tokenSecret": "CHANGE-ME-TO-A-RANDOM-SECRET"
  },
  "storage": {
    "dbPath": "./data/openfi.db"
  },
  "agentPool": {
    "fixedKey": "CHANGE-ME-TO-A-64-CHAR-HEX-KEY-0123456789abcdef0123456789abcdef"
  },
  "hyperliquid": {
    "baseURL": "https://api.hyperliquid.xyz"
  },
  "log": {
    "dir": "./logs",
    "level": "info",
    "maxSize": 100,
    "maxFiles": 10,
    "console": true
  }
}
CONF
fi

# Create start script
echo "[release] creating start script..."
cat > "${RELEASE_DIR}/start.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"
exec ./openfi-server -config ./config/config.json
EOF
chmod +x "${RELEASE_DIR}/start.sh"

# Copy install script
echo "[release] copying install script..."
cp "${ROOT_DIR}/install.sh" "${RELEASE_DIR}/install.sh"
chmod +x "${RELEASE_DIR}/install.sh"

# Package the release
echo "[release] packaging..."
cd "${DIST_DIR}"
tar -czf "${RELEASE_PACKAGE}" "${RELEASE_NAME}"
rm -rf "${RELEASE_DIR}"

echo "[release] done!"
echo "[release] package: ${RELEASE_PACKAGE}"
echo "[release] size: $(du -h "${RELEASE_PACKAGE}" | cut -f1)"
