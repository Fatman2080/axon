#!/usr/bin/env bash
#
# OpenFi Server — systemd install script
#
# Usage:
#   sudo ./install.sh              # install with defaults
#   sudo ./install.sh --uninstall  # remove service and user
#
# After install:
#   sudo systemctl status openfi
#   sudo journalctl -u openfi -f
#
set -euo pipefail

SERVICE_NAME="openfi"
SERVICE_USER="openfi"
INSTALL_DIR="/opt/openfi"
BACKUP_DIR="/opt/openfi-backups"
BACKUP_KEEP=10
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;32m[install]\033[0m $*"; }
warn()  { echo -e "\033[1;33m[install]\033[0m $*"; }
error() { echo -e "\033[1;31m[install]\033[0m $*"; exit 1; }

need_root() {
  [[ $EUID -eq 0 ]] || error "please run with sudo or as root"
}

# ---------------------------------------------------------------------------
# Uninstall
# ---------------------------------------------------------------------------
do_uninstall() {
  need_root
  info "stopping service..."
  systemctl stop "${SERVICE_NAME}" 2>/dev/null || true
  systemctl disable "${SERVICE_NAME}" 2>/dev/null || true

  info "removing unit file..."
  rm -f "${UNIT_FILE}"
  systemctl daemon-reload

  info "removing install directory ${INSTALL_DIR}..."
  rm -rf "${INSTALL_DIR}"

  info "removing user ${SERVICE_USER}..."
  userdel "${SERVICE_USER}" 2>/dev/null || true

  info "uninstall complete"
  exit 0
}

[[ "${1:-}" == "--uninstall" ]] && do_uninstall

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------
need_root

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Stop running service before install
if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
    read -rp "[install] ${SERVICE_NAME} is running. Stop it to proceed? [y/N] " ans
    case "${ans}" in
        [yY]*) info "stopping ${SERVICE_NAME}..."; systemctl stop "${SERVICE_NAME}" ;;
        *)     error "install cancelled — service is still running" ;;
    esac
fi

# Backup existing installation (exclude logs/)
if [[ -d "${INSTALL_DIR}" ]]; then
  mkdir -p "${BACKUP_DIR}"
  backup_file="${BACKUP_DIR}/openfi-$(date +%Y%m%d-%H%M%S).tar.gz"
  info "backing up ${INSTALL_DIR} → ${backup_file} ..."
  tar -czf "${backup_file}" -C /opt --exclude='openfi/logs' openfi

  # Rotate: keep only the newest BACKUP_KEEP backups
  backup_count=$(ls -1t "${BACKUP_DIR}"/openfi-*.tar.gz 2>/dev/null | wc -l)
  if (( backup_count > BACKUP_KEEP )); then
    ls -1t "${BACKUP_DIR}"/openfi-*.tar.gz | tail -n +"$(( BACKUP_KEEP + 1 ))" | xargs rm -f
  fi
  info "backup complete (keeping latest ${BACKUP_KEEP})"
fi

# Verify we are inside a release package
[[ -f "${SCRIPT_DIR}/openfi-server" ]] || error "openfi-server binary not found in ${SCRIPT_DIR}"
[[ -d "${SCRIPT_DIR}/config" ]]        || error "config/ directory not found in ${SCRIPT_DIR}"
[[ -d "${SCRIPT_DIR}/assets" ]]        || error "assets/ directory not found in ${SCRIPT_DIR}"

# 1. Create system user (no login shell, no home dir)
if ! id "${SERVICE_USER}" &>/dev/null; then
  info "creating system user ${SERVICE_USER}..."
  useradd --system --no-create-home --shell /usr/sbin/nologin "${SERVICE_USER}"
else
  info "user ${SERVICE_USER} already exists"
fi

# 2. Copy files
info "installing to ${INSTALL_DIR}..."
mkdir -p "${INSTALL_DIR}"
cp    "${SCRIPT_DIR}/openfi-server" "${INSTALL_DIR}/openfi-server"
cp    "${SCRIPT_DIR}/start.sh"      "${INSTALL_DIR}/start.sh"
cp -r "${SCRIPT_DIR}/assets"        "${INSTALL_DIR}/assets"

# Config — don't overwrite if already exists (preserve user edits)
if [[ ! -d "${INSTALL_DIR}/config" ]]; then
  cp -r "${SCRIPT_DIR}/config" "${INSTALL_DIR}/config"
fi

# Data dir
mkdir -p "${INSTALL_DIR}/data"

# Logs dir
mkdir -p "${INSTALL_DIR}/logs"

# 3. Permissions
chmod 755 "${INSTALL_DIR}/openfi-server"
chmod 755 "${INSTALL_DIR}/start.sh"
chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"

# 4. Create systemd unit
info "creating systemd service..."
cat > "${UNIT_FILE}" << EOF
[Unit]
Description=OpenFi Server
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/openfi-server -config ${INSTALL_DIR}/config/config.json
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}/data ${INSTALL_DIR}/config ${INSTALL_DIR}/logs
PrivateTmp=true

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${SERVICE_NAME}

[Install]
WantedBy=multi-user.target
EOF

# 5. Enable and start
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl start "${SERVICE_NAME}"

info "install complete!"
info ""
info "  service status:  sudo systemctl status ${SERVICE_NAME}"
info "  view logs:       sudo journalctl -u ${SERVICE_NAME} -f"
info "  edit config:     sudo -u ${SERVICE_USER} vi ${INSTALL_DIR}/config/config.json"
info "  restart:         sudo systemctl restart ${SERVICE_NAME}"
info "  uninstall:       sudo ${INSTALL_DIR}/install.sh --uninstall"
info ""
info "  default port:    9333 (configure nginx reverse proxy to this port)"
