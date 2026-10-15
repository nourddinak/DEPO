#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# DEPO Central Server - One-Line Automated VPS Installer
# Installs the DEPO control plane binary, data directory, and systemd service.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/<user>/depo/main/scripts/install.sh | sudo bash
#   curl -sSL https://raw.githubusercontent.com/<user>/depo/main/scripts/install.sh | sudo bash -s -- --port 8080 --domain depo.example.com
# ==============================================================================

PORT="8080"
DATA_DIR="/var/lib/depo"
DOMAIN=""
VERSION="latest"
GITHUB_REPO="depo-platform/depo"
INSTALL_BIN="/usr/local/bin/depo"
SERVICE_NAME="depo"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --port)
      PORT="$2"
      shift 2
      ;;
    --data)
      DATA_DIR="$2"
      shift 2
      ;;
    --domain)
      DOMAIN="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --repo)
      GITHUB_REPO="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

if [[ $EUID -ne 0 ]]; then
  echo "Error: This installer must be run as root (or with sudo)." >&2
  exit 1
fi

echo "=========================================================="
echo "          DEPO Central Control Plane Installer           "
echo "=========================================================="
echo "Install Port:     $PORT"
echo "Data Directory:   $DATA_DIR"
if [[ -n "$DOMAIN" ]]; then
  echo "Public Domain:    https://$DOMAIN"
fi

# 1. Prerequisites Check
echo "[1/5] Checking system prerequisites..."
if ! command -v curl &>/dev/null; then
  apt-get update -qq && apt-get install -y -qq curl
fi
if ! command -v tar &>/dev/null; then
  apt-get install -y -qq tar gzip
fi

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) BIN_ARCH="amd64" ;;
  aarch64|arm64) BIN_ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# 2. Directory Initialization
echo "[2/5] Initializing data directory at $DATA_DIR..."
mkdir -p "$DATA_DIR" "$DATA_DIR/artifacts" /usr/local/bin
chmod 700 "$DATA_DIR"

# 3. Binary Installation
echo "[3/5] Installing depo binary for linux-$BIN_ARCH..."

# Strategy A: Check if a local binary is provided in the current directory
if [[ -f "./depo" ]]; then
  cp ./depo "$INSTALL_BIN"
  echo "Installed local depo binary."
# Strategy B: Download from GitHub Releases
else
  RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/depo-linux-${BIN_ARCH}"
  if [[ "$VERSION" != "latest" ]]; then
    RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/depo-linux-${BIN_ARCH}"
  fi

  echo "Fetching release binary: $RELEASE_URL"
  if curl -sSL -f -o "$INSTALL_BIN" "$RELEASE_URL"; then
    echo "Successfully downloaded depo binary."
  else
    # Strategy C: If Go is installed locally, attempt build as fallback
    if command -v go &>/dev/null && [[ -f "./cmd/depo/main.go" ]]; then
      echo "Compiling depo from source..."
      CGO_ENABLED=0 go build -ldflags="-s -w" -o "$INSTALL_BIN" ./cmd/depo
    else
      echo "Notice: Could not download pre-compiled binary from GitHub release yet."
      echo "If you are running from the source checkout, run: go build -o /usr/local/bin/depo ./cmd/depo"
      if [[ ! -f "$INSTALL_BIN" ]]; then
        exit 1
      fi
    fi
  fi
fi

chmod 755 "$INSTALL_BIN"

# 4. systemd Service Setup
echo "[4/5] Installing systemd service ($SERVICE_NAME.service)..."
cat <<EOF > /etc/systemd/system/${SERVICE_NAME}.service
[Unit]
Description=DEPO Central Control Plane API & Dashboard
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${DATA_DIR}
ExecStart=${INSTALL_BIN} --data ${DATA_DIR} --port ${PORT}
Restart=always
RestartSec=5s
LimitNOFILE=65536
Environment="DEPO_DATA_DIR=${DATA_DIR}"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ${SERVICE_NAME}.service
systemctl restart ${SERVICE_NAME}.service

# 5. Optional Caddy Reverse Proxy Setup
if [[ -n "$DOMAIN" ]]; then
  echo "[5/5] Configuring Caddy reverse proxy for $DOMAIN..."
  if ! command -v caddy &>/dev/null; then
    apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
    curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg --yes
    curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
    apt-get update -qq && apt-get install -y -qq caddy
  fi

  CADDYFILE_SNIPPET="/etc/caddy/Caddyfile"
  if [[ -f "$CADDYFILE_SNIPPET" ]]; then
    if ! grep -q "$DOMAIN" "$CADDYFILE_SNIPPET"; then
      cat <<EOF >> "$CADDYFILE_SNIPPET"

# DEPO Central Control Plane
$DOMAIN {
    reverse_proxy 127.0.0.1:$PORT
}
EOF
      systemctl reload caddy || systemctl restart caddy
    fi
  fi
else
  echo "[5/5] Skipping reverse proxy (no --domain specified)."
fi

# Get Server Public IP
PUBLIC_IP=$(curl -sSL -4 https://icanhazip.com || curl -sSL -4 https://api.ipify.org || echo "your-server-ip")

echo "=========================================================="
echo "    DEPO Central Server Installed & Running!              "
echo "=========================================================="
if [[ -n "$DOMAIN" ]]; then
  echo " Dashboard URL: https://${DOMAIN}"
else
  echo " Dashboard URL: http://${PUBLIC_IP}:${PORT}"
fi
echo " Service Name:  ${SERVICE_NAME}.service"
echo " Status Check:  systemctl status ${SERVICE_NAME}"
echo " Live Logs:     journalctl -u ${SERVICE_NAME} -f"
echo " Data Path:     ${DATA_DIR}"
echo ""
echo "Next step: Open the Dashboard URL in your browser to complete"
echo "the initial owner account setup wizard."
echo "=========================================================="
