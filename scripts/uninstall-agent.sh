#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# DEPO Edge Deployment Agent - Full VPS Uninstaller
# Completely removes the DEPO agent binary, config, systemd service, and tokens.
#
# Usage:
#   curl -sSL https://depo.example.com/uninstall-agent.sh | sudo bash
#   curl -sSL https://depo.example.com/uninstall-agent.sh | sudo bash -s -- --purge-www
# ==============================================================================

SERVICE_NAME="depo-agent"
INSTALL_BIN="/usr/local/bin/depo-agent"
CONFIG_DIR="/etc/depo-agent"
DATA_DIR="/var/lib/depo-agent"
PURGE_WWW=false
FORCE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --purge-www)
      PURGE_WWW=true
      shift
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
echo "          DEPO Deployment Agent Uninstaller               "
echo "=========================================================="
echo "Service Name:    $SERVICE_NAME.service"
echo "Binary Location: $INSTALL_BIN"
echo "Config Path:     $CONFIG_DIR"
echo "Runtime Path:    $DATA_DIR"
echo ""

# 1. Stop and Disable systemd service
echo "[1/4] Stopping and disabling agent service..."
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
  systemctl stop "$SERVICE_NAME" || true
  echo "Stopped $SERVICE_NAME.service."
fi

if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
  systemctl disable "$SERVICE_NAME" || true
  echo "Disabled $SERVICE_NAME.service."
fi

# 2. Remove systemd service file
echo "[2/4] Removing systemd unit..."
if [[ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
  rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
  systemctl daemon-reload || true
  systemctl reset-failed 2>/dev/null || true
  echo "Removed /etc/systemd/system/${SERVICE_NAME}.service."
fi

# 3. Remove Binary
echo "[3/4] Removing agent binary..."
if [[ -f "$INSTALL_BIN" ]]; then
  rm -f "$INSTALL_BIN"
  echo "Removed $INSTALL_BIN."
fi

# 4. Remove Agent Configuration & Runtime State
echo "[4/4] Removing configuration and credentials..."
if [[ -d "$CONFIG_DIR" ]]; then
  rm -rf "$CONFIG_DIR"
  echo "Removed $CONFIG_DIR."
fi

if [[ -d "$DATA_DIR" ]]; then
  rm -rf "$DATA_DIR"
  echo "Removed $DATA_DIR."
fi

# Optional: Cleanup web releases in /var/www/depo if requested
if [[ "$PURGE_WWW" == true ]]; then
  echo "Purging /var/www/depo static release directories..."
  if [[ -d "/var/www/depo" ]]; then
    rm -rf /var/www/depo/* /var/www/depo/.[!.]* /var/www/depo/..?* 2>/dev/null || true
    echo "Cleaned /var/www/depo."
  fi
else
  echo "Notice: Static web releases in /var/www/depo were preserved."
fi

echo "=========================================================="
echo "    DEPO Agent has been completely uninstalled.           "
echo "=========================================================="
