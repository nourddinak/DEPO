#!/usr/bin/env bash
set -euo pipefail

# DEPO VPS Agent One-Line Automated Installer
# Usage:
#   curl -sSL https://depo.example.com/install-agent.sh | sudo bash -s -- --server https://depo.example.com --token depo_enr_xxxx

SERVER_URL=""
ENROLL_TOKEN=""
AGENT_VERSION="1.0.0"
CONFIG_PATH="/etc/depo-agent/config.json"
INSTALL_BIN="/usr/local/bin/depo-agent"
SERVICE_NAME="depo-agent"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server)
      SERVER_URL="$2"
      shift 2
      ;;
    --token)
      ENROLL_TOKEN="$2"
      shift 2
      ;;
    --version)
      AGENT_VERSION="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

if [[ -z "$SERVER_URL" || -z "$ENROLL_TOKEN" ]]; then
  echo "Error: Both --server <URL> and --token <ENROLL_TOKEN> are required."
  exit 1
fi

if [[ $EUID -ne 0 ]]; then
  echo "Error: This installer must be run as root (or with sudo)."
  exit 1
fi

echo "=========================================================="
echo "           DEPO VPS Deployment Agent Installer            "
echo "=========================================================="
echo "Target Central Server: $SERVER_URL"

# 1. Ensure required tools exist (curl, tar, caddy)
echo "[1/6] Checking prerequisites..."
if ! command -v curl &>/dev/null; then
  apt-get update -qq && apt-get install -y -qq curl
fi
if ! command -v tar &>/dev/null; then
  apt-get install -y -qq tar gzip
fi

# Ensure Caddy is installed
if ! command -v caddy &>/dev/null; then
  echo "Caddy not detected. Installing official Caddy on Ubuntu..."
  apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg --yes
  curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
  apt-get update -qq && apt-get install -y -qq caddy
  systemctl enable --now caddy
fi

# 2. Setup Directories & Permissions
echo "[2/6] Configuring directories..."
mkdir -p /etc/depo-agent /var/lib/depo-agent /var/www /etc/caddy
chmod 750 /etc/depo-agent /var/lib/depo-agent
chown -R root:www-data /var/www || true
chmod -R 775 /var/www || true

# 3. Download or install depo-agent binary
echo "[3/6] Installing depo-agent binary..."
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) AGENT_ARCH="amd64" ;;
  aarch64|arm64) AGENT_ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download binary from central server if available, or fetch release
DOWNLOAD_URL="${SERVER_URL}/downloads/depo-agent-linux-${AGENT_ARCH}"
if curl -sSL -f -o "$INSTALL_BIN" "$DOWNLOAD_URL" 2>/dev/null; then
  echo "Downloaded agent binary from central server."
else
  echo "Downloading agent from release assets..."
  GITHUB_RELEASE_URL="https://github.com/nourddinak/DEPO/releases/latest/download/depo-agent-linux-${AGENT_ARCH}"
  if curl -sSL -f -o "$INSTALL_BIN" "$GITHUB_RELEASE_URL" 2>/dev/null; then
    echo "Downloaded agent binary from GitHub release."
  elif [[ -f "./depo-agent" ]]; then
    cp ./depo-agent "$INSTALL_BIN"
  else
    echo "Pre-compiled agent binary not found. Compiling from source..."
    BUILD_DIR=$(mktemp -d /tmp/depo-agent-build.XXXXXX)
    if ! command -v git &>/dev/null; then
      apt-get update -qq && apt-get install -y -qq git
    fi
    git clone --depth 1 "https://github.com/nourddinak/DEPO.git" "$BUILD_DIR"
    if ! command -v go &>/dev/null; then
      apt-get update -qq && apt-get install -y -qq golang-go || true
      if ! command -v go &>/dev/null; then
        GO_TARBALL_URL="https://go.dev/dl/go1.24.0.linux-${AGENT_ARCH}.tar.gz"
        curl -sSL "$GO_TARBALL_URL" | tar -C /usr/local -xz
        export PATH="/usr/local/go/bin:$PATH"
      fi
    fi
    (cd "$BUILD_DIR" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "$INSTALL_BIN" ./cmd/depo-agent)
    rm -rf "$BUILD_DIR"
    echo "Compiled and installed depo-agent binary."
  fi
fi

chmod +x "$INSTALL_BIN" || true

# 4. Enroll Server
echo "[4/6] Enrolling server with central control plane..."
"$INSTALL_BIN" enroll --server "$SERVER_URL" --token "$ENROLL_TOKEN" --config "$CONFIG_PATH"

# 5. Create systemd Service Unit
echo "[5/6] Creating systemd service unit..."
cat <<EOF > /etc/systemd/system/${SERVICE_NAME}.service
[Unit]
Description=DEPO Deployment Agent
After=network.target network-online.target caddy.service
Wants=network-online.target

[Service]
Type=simple
User=root
Group=www-data
ExecStart=${INSTALL_BIN} run --config ${CONFIG_PATH}
Restart=always
RestartSec=5s
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 6. Enable and Start Service
echo "[6/6] Starting agent service..."
systemctl daemon-reload
systemctl enable ${SERVICE_NAME}.service
systemctl restart ${SERVICE_NAME}.service

echo "=========================================================="
echo "  DEPO Agent installed and running successfully!          "
echo "  Status: systemctl status ${SERVICE_NAME}                "
echo "  Logs:   journalctl -u ${SERVICE_NAME} -f                "
echo "=========================================================="
