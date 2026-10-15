#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# DEPO Central Server - Full VPS Uninstaller
# Completely removes the DEPO control plane binary, systemd service, and optionally data.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/nourddinak/DEPO/main/scripts/uninstall.sh | sudo bash
#   curl -sSL https://raw.githubusercontent.com/nourddinak/DEPO/main/scripts/uninstall.sh | sudo bash -s -- --purge-data
# ==============================================================================

SERVICE_NAME="depo"
INSTALL_BIN="/usr/local/bin/depo"
DATA_DIR="/var/lib/depo"
PURGE_DATA=false
FORCE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --purge-data)
      PURGE_DATA=true
      shift
      ;;
    --keep-data)
      PURGE_DATA=false
      FORCE=true
      shift
      ;;
    --data)
      DATA_DIR="$2"
      shift 2
      ;;
    -f|--force)
      FORCE=true
      shift
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

if [[ $EUID -ne 0 ]]; then
  echo "Error: This uninstaller must be run as root (or with sudo)." >&2
  exit 1
fi

echo "=========================================================="
echo "          DEPO Central Server Uninstaller                 "
echo "=========================================================="
echo "Service Name:    $SERVICE_NAME.service"
echo "Binary Location: $INSTALL_BIN"
echo "Data Directory:  $DATA_DIR"
echo ""

# Confirm data purge if not in force/purge mode and running interactively
if [[ "$PURGE_DATA" == false && "$FORCE" == false && -t 0 ]]; then
  read -r -p "Do you want to permanently delete all database files and secrets at $DATA_DIR? [y/N]: " confirm_purge
  if [[ "$confirm_purge" =~ ^[Yy]$ ]]; then
    PURGE_DATA=true
  fi
fi

# 1. Stop and Disable systemd service
echo "[1/4] Stopping and disabling systemd service..."
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
  systemctl stop "$SERVICE_NAME" || true
  echo "Stopped $SERVICE_NAME.service."
fi

if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
  systemctl disable "$SERVICE_NAME" || true
  echo "Disabled $SERVICE_NAME.service."
fi

# 2. Remove systemd service file
echo "[2/4] Removing systemd unit configuration..."
if [[ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
  rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
  systemctl daemon-reload || true
  systemctl reset-failed 2>/dev/null || true
  echo "Removed /etc/systemd/system/${SERVICE_NAME}.service."
fi

# 3. Remove Binary
echo "[3/4] Removing installed binary..."
if [[ -f "$INSTALL_BIN" ]]; then
  rm -f "$INSTALL_BIN"
  echo "Removed $INSTALL_BIN."
fi

# Clean up Caddy reverse proxy entry if it exists
CADDYFILE="/etc/caddy/Caddyfile"
if [[ -f "$CADDYFILE" ]] && grep -q "# DEPO Central Control Plane" "$CADDYFILE"; then
  echo "Cleaning DEPO entries from $CADDYFILE..."
  sed -i '/# DEPO Central Control Plane/,+3d' "$CADDYFILE" || true
  if command -v caddy &>/dev/null; then
    systemctl reload caddy 2>/dev/null || true
  fi
fi

# 4. Handle Data Directory
echo "[4/4] Finalizing data cleanup..."
if [[ "$PURGE_DATA" == true ]]; then
  if [[ -d "$DATA_DIR" ]]; then
    rm -rf "$DATA_DIR"
    echo "Purged data directory: $DATA_DIR"
  fi
else
  if [[ -d "$DATA_DIR" ]]; then
    echo "Notice: Data directory preserved at $DATA_DIR."
    echo "To remove database records and keys manually later, run: sudo rm -rf $DATA_DIR"
  fi
fi

echo "=========================================================="
echo "    DEPO Central Server has been completely uninstalled.  "
echo "=========================================================="
