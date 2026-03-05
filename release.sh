#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
RELEASE_DIR="${DIST_DIR}/openfi-release"
VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
RELEASE_NAME="openfi-${VERSION}"
RELEASE_PACKAGE="${DIST_DIR}/${RELEASE_NAME}.tar.gz"

echo "[release] starting release build..."
echo "[release] version: ${VERSION}"

# Clean and prepare directories
echo "[release] preparing directories..."
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}/config"
mkdir -p "${RELEASE_DIR}/data"
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

# Copy config files
echo "[release] copying config files..."
if [[ -f "${ROOT_DIR}/config/config.prod.json" ]]; then
  cp "${ROOT_DIR}/config/config.prod.json" "${RELEASE_DIR}/config/config.json"
else
  cp "${ROOT_DIR}/config/config.sample.json" "${RELEASE_DIR}/config/config.json"
fi

# Create start script
echo "[release] creating start script..."
cat > "${RELEASE_DIR}/start.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

./openfi-server -config ./config/config.json
EOF
chmod +x "${RELEASE_DIR}/start.sh"

# Create README
echo "[release] creating README..."
cat > "${RELEASE_DIR}/README.md" << 'EOF'
# OpenFI Server Release Package

## Directory Structure
- `openfi-server`: Server binary
- `config/`: Configuration files
- `data/`: Data directory (will be created on first run)
- `assets/www/`: Frontend website files
- `assets/admin/`: Admin panel files
- `start.sh`: Start script

## Quick Start
1. Edit `config/config.json` to configure your server
2. Run `./start.sh` to start the server

## Requirements
- Linux x64 system
- Port access as configured in config.json
EOF

# Package the release
echo "[release] packaging release..."
cd "${DIST_DIR}"
tar -czf "${RELEASE_PACKAGE}" -C "${DIST_DIR}" "openfi-release"
mv "${DIST_DIR}/openfi-release" "${DIST_DIR}/${RELEASE_NAME}"
tar -czf "${RELEASE_PACKAGE}" -C "${DIST_DIR}" "${RELEASE_NAME}"
rm -rf "${DIST_DIR}/${RELEASE_NAME}"

echo "[release] ✓ release build completed!"
echo "[release] package: ${RELEASE_PACKAGE}"
echo "[release] size: $(du -h "${RELEASE_PACKAGE}" | cut -f1)"
