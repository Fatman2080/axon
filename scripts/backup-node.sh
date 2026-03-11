#!/bin/bash
set -euo pipefail

# ============================================================
# Axon Node Backup Script
#
# Creates a timestamped backup of node data, config, and keys.
# Usage: bash scripts/backup-node.sh [--home DIR] [--output DIR]
#
# IMPORTANT: Stop the node before backing up to ensure consistency:
#   sudo systemctl stop axond
#   bash scripts/backup-node.sh
#   sudo systemctl start axond
# ============================================================

AXON_HOME="${AXON_HOME:-$HOME/.axond}"
BACKUP_DIR="${BACKUP_DIR:-$HOME/axon-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

while [[ $# -gt 0 ]]; do
  case $1 in
    --home)   AXON_HOME="$2"; shift 2 ;;
    --output) BACKUP_DIR="$2"; shift 2 ;;
    *) echo "Unknown: $1"; exit 1 ;;
  esac
done

if [ ! -d "$AXON_HOME" ]; then
    echo "ERROR: Axon home directory not found: $AXON_HOME"
    exit 1
fi

BACKUP_FILE="$BACKUP_DIR/axon-backup-${TIMESTAMP}.tar.gz"
mkdir -p "$BACKUP_DIR"

echo "=== Axon Node Backup ==="
echo "  Home:   $AXON_HOME"
echo "  Output: $BACKUP_FILE"
echo ""

PIDS=$(pgrep -f "axond start" 2>/dev/null || true)
if [ -n "$PIDS" ]; then
    echo "WARNING: axond appears to be running (PID: $PIDS)."
    echo "         For a consistent backup, stop the node first:"
    echo "         sudo systemctl stop axond"
    echo ""
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Backup cancelled."
        exit 0
    fi
fi

echo "Backing up..."

INCLUDE_PATHS=()

if [ -d "$AXON_HOME/config" ]; then
    INCLUDE_PATHS+=("config")
fi

if [ -d "$AXON_HOME/data" ]; then
    INCLUDE_PATHS+=("data")
fi

if [ -d "$AXON_HOME/keyring-file" ]; then
    INCLUDE_PATHS+=("keyring-file")
fi

if [ -d "$AXON_HOME/keyring-test" ]; then
    INCLUDE_PATHS+=("keyring-test")
fi

for dir in "$AXON_HOME"/keyring-*; do
    if [ -d "$dir" ]; then
        base=$(basename "$dir")
        already=false
        for p in "${INCLUDE_PATHS[@]}"; do
            if [ "$p" = "$base" ]; then already=true; break; fi
        done
        if [ "$already" = false ]; then
            INCLUDE_PATHS+=("$base")
        fi
    fi
done

if [ ${#INCLUDE_PATHS[@]} -eq 0 ]; then
    echo "ERROR: No directories found to back up in $AXON_HOME"
    exit 1
fi

tar -czf "$BACKUP_FILE" -C "$AXON_HOME" "${INCLUDE_PATHS[@]}"

SIZE=$(du -sh "$BACKUP_FILE" | cut -f1)
echo ""
echo "=== Backup Complete ==="
echo "  File: $BACKUP_FILE"
echo "  Size: $SIZE"
echo "  Contents: ${INCLUDE_PATHS[*]}"
echo ""
echo "To restore:"
echo "  sudo systemctl stop axond"
echo "  tar -xzf $BACKUP_FILE -C $AXON_HOME"
echo "  sudo systemctl start axond"

OLD_BACKUPS=$(find "$BACKUP_DIR" -name "axon-backup-*.tar.gz" -mtime +30 2>/dev/null | wc -l)
if [ "$OLD_BACKUPS" -gt 0 ]; then
    echo ""
    echo "TIP: $OLD_BACKUPS backup(s) older than 30 days found in $BACKUP_DIR"
    echo "     Clean up with: find $BACKUP_DIR -name 'axon-backup-*.tar.gz' -mtime +30 -delete"
fi
