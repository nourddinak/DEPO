# DEPO: Multi-VPS Static Frontend Deployment Platform

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)](https://react.dev)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](docker-compose.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

DEPO is a modern, self-hosted, lightweight deployment platform and control plane engineered to host, automate, and orchestrate static frontend applications across fleets of Ubuntu/Debian VPS servers. It links directly with GitHub Actions, delivers cryptographically verified tar.gz build artifacts over outbound-only connections, and executes atomic zero-downtime releases with automated rollbacks and Caddy SSL management.

---

## Highlights

- **Zero-Port Outbound Agents**: Target VPS edge servers run a lightweight Go daemon that long-polls the central control plane over secure HTTPS. No incoming firewall ports or SSH keys are ever opened to the internet.
- **Single Unified Binary**: The central server compiles into a single static Go binary with the complete React 19 + TypeScript dashboard embedded (`embed.FS`). No external Node runtime or reverse proxy required on the control plane.
- **Atomic Zero-Downtime Releases**: Deploys web bundles to `/var/www/<project>/releases/<release-id>` and executes an atomic same-filesystem symlink switch to `/var/www/<project>/current`.
- **Automated Caddy Management**: Idempotently generates and provisions reverse proxy blocks in `/etc/caddy/Caddyfile`, validates syntax with `caddy validate`, reloads via Caddy's dynamic runtime API, and restores backups automatically on failure.
- **Dual-Mode CI/CD Authentication**: Supports both zero-secret GitHub Actions OIDC JWT claim verification and scoped bearer deploy tokens.
- **Encrypted Secrets & Environment Variables**: Encrypts all environment variables at rest using AES-256-GCM with hardware-backed authenticated encryption.
- **Ephemeral Pull Request Previews**: Spawns isolated preview environments for pull requests with automatic wildcard routing and automated TTL cleanup upon merge or closure.
- **Live Telemetry & Alerts**: Streams live CPU, RAM, and Disk metrics via Server-Sent Events (SSE) and dispatches real-time push notifications via `ntfy` with automatic secret redaction.
- **Hardened Security**: Argon2id password hashing, optional TOTP two-factor authentication with recovery codes, strict Zip-Slip path traversal rejection, and non-root execution permissions.

---

## Architecture

```
                                  +------------------------------------+
                                  |        GitHub Actions CI/CD        |
                                  |  - Builds React/Vite/Vue/Astro/etc |
                                  |  - Packages tar.gz + SHA256        |
                                  |  - Authenticates via OIDC / Token  |
                                  +-----------------+------------------+
                                                    |
                                                    | POST /api/v1/projects/:id/deploy
                                                    v
+-----------------------------+    +------------------------------------+
|   Dashboard UI (SPA)        |    |        DEPO Central Server         |
|   React 19 + TypeScript     |<==>|  - Go HTTP Control Plane (:8080)   |
|   - Real-time updates (SSE) |REST|  - Pure-Go SQLite WAL Engine       |
|   - Zero-external assets    |&SSE|  - AES-256-GCM Cryptographic Store |
+-----------------------------+    |  - Job Queue with Leases           |
                                   |  - Artifact Storage & Verification |
                                   |  - Serves Installer Scripts        |
                                   +-----------------+------------------+
                                                     ^
                                                     | Outbound Long Polling
                                                     | (GET /api/v1/agent/jobs/poll)
                                                     | Heartbeat & Metrics (POST)
                                                     v
                                   +------------------------------------+
                                   |       DEPO VPS Agent (Edge)        |
                                   |  - Runs as systemd service         |
                                   |  - Atomic release symlink swap     |
                                   |  - Caddy validation & auto-reload  |
                                   |  - Local health checks & rollback  |
                                   |  - Real-time host telemetry        |
                                   +-----------------+------------------+
                                                     |
                                                     v
                                   +------------------------------------+
                                   |       Caddy Web Server             |
                                   |  - Serves /var/www/<project>/curr  |
                                   |  - Automatic HTTPS / TLS (Let's    |
                                   |    Encrypt / ZeroSSL)              |
                                   +------------------------------------+
```

---

## Deployment Options

Choose the deployment method that fits your infrastructure best:

### Option 1: 1-Line Automated VPS Installer (Recommended)

Run this command on any fresh Ubuntu/Debian VPS (x86_64 or ARM64):

```bash
curl -sSL https://raw.githubusercontent.com/depo-platform/depo/main/scripts/install.sh | sudo bash
```

To configure custom ports or enable automated Caddy reverse proxy with SSL:
```bash
curl -sSL https://raw.githubusercontent.com/depo-platform/depo/main/scripts/install.sh | sudo bash -s -- --port 8080 --domain depo.example.com
```

The script will:
1. Detect architecture (`amd64` / `arm64`) and download the latest release binary.
2. Initialize `/var/lib/depo` for SQLite storage and encrypted master keys.
3. Install and start the `depo.service` systemd daemon.
4. Optionally configure Caddy to provide instant automatic HTTPS.

---

### Option 2: Docker Compose

For containerized environments:

1. Clone or download `docker-compose.yml`:
```bash
curl -sSL -O https://raw.githubusercontent.com/depo-platform/depo/main/docker-compose.yml
```

2. Start the container in detached mode:
```bash
docker compose up -d
```

3. Check logs and status:
```bash
docker compose ps
docker compose logs -f
```

The dashboard will be accessible at `http://<your-vps-ip>:8080`. Data is stored persistently in the `depo_data` Docker volume.

---

### Option 3: Compile from Source / Single Binary

Prerequisites: Go 1.24+ and Node 22+ (only required if building frontend from source).

```bash
# 1. Clone the repository
git clone https://github.com/depo-platform/depo.git
cd depo

# 2. Build the control plane (pre-built frontend assets are included in cmd/depo/dist)
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/depo ./cmd/depo

# 3. Run the central server
./bin/depo --data ./data --port 8080
```

---

## Getting Started

### 1. Initial Setup
1. Open `http://<your-server-ip>:8080` (or `https://depo.example.com`) in your web browser.
2. On first launch, the setup wizard will prompt you to create the initial **Administrator Account**.
3. Optionally configure TOTP Two-Factor Authentication (2FA) and backup recovery codes.

---

### 2. Enrolling a VPS Edge Agent
To attach an edge server to deploy websites onto:

1. In the DEPO dashboard, navigate to **VPS Fleet** -> **Enroll New Server**.
2. Enter a friendly name (e.g., `web-vps-frankfurt-01`) and click **Generate Enrollment Command**.
3. SSH into your edge VPS and execute the generated one-liner:

```bash
curl -sSL http://<depo-server>:8080/install-agent.sh | sudo bash -s -- --server http://<depo-server>:8080 --token depo_enr_xxxxxxxxxxxx
```

4. The agent will:
   - Auto-install and start Caddy (if not already present).
   - Verify network connectivity to the central server.
   - Configure `/etc/systemd/system/depo-agent.service`.
   - Appear as **Online** on your DEPO dashboard within 5 seconds, streaming live CPU, memory, and disk telemetry.

---

### 3. Deploying Your First Project

1. In the DEPO dashboard, click **New Project**.
2. Fill in:
   - **Project Name**: `my-vite-app`
   - **Domain**: `my-app.example.com`
   - **Target VPS**: Select your enrolled server.
   - **Framework**: Choose `Vite`, `Next.js (Export)`, `Astro`, `Vue`, `Angular`, or `Static HTML`.
3. Point your domain's DNS A/AAAA record to your edge VPS IP address.
4. In your GitHub repository, create `.github/workflows/depo-deploy.yml`:

```yaml
name: Deploy to DEPO

on:
  push:
    branches: [ "main" ]

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Repository
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: 'npm'

      - name: Install & Build
        run: |
          npm ci
          npm run build

      - name: Package Artifact
        run: |
          cd dist
          tar -czf ../artifact.tar.gz .
          cd ..

      - name: Deploy to DEPO
        run: |
          curl -X POST "https://depo.example.com/api/v1/projects/${{ secrets.DEPO_PROJECT_ID }}/deploy" \
            -H "Authorization: Bearer ${{ secrets.DEPO_DEPLOY_TOKEN }}" \
            -F "artifact=@artifact.tar.gz" \
            -F "commit_sha=${{ github.sha }}" \
            -F "commit_message=${{ github.event.head_commit.message }}"
```

5. Push a commit to `main`. DEPO handles artifact storage, leases the job to the agent, extracts the bundle safely, executes zero-downtime symlink switching, updates Caddy configuration, and reports live status logs back to your dashboard!

---

## Pull Request Previews

DEPO natively supports on-demand preview environments for pull requests:
- Set up a wildcard DNS record (e.g. `*.preview.example.com` -> edge VPS IP).
- Configure GitHub PR Webhooks in project settings.
- When a PR is opened, DEPO automatically generates an isolated preview deployment at `pr-<number>.<project>.preview.example.com`.
- When the PR is closed or merged, the preview environment, directories, and Caddy routing rules are cleaned up automatically.

---

## Uninstallation & Cleanup

DEPO provides automated, clean uninstaller scripts for both the central control plane and edge agents:

### 1. Uninstall Central Server
To remove the DEPO control plane service and binary from your VPS:
```bash
curl -sSL https://raw.githubusercontent.com/depo-platform/depo/main/scripts/uninstall.sh | sudo bash
```
To also permanently delete all SQLite databases, encryption keys, and artifacts located in `/var/lib/depo`:
```bash
curl -sSL https://raw.githubusercontent.com/depo-platform/depo/main/scripts/uninstall.sh | sudo bash -s -- --purge-data
```

### 2. Uninstall VPS Edge Agent
To completely remove the agent daemon, credentials, and systemd service from a target VPS:
```bash
# Served directly from your DEPO control plane:
curl -sSL https://depo.example.com/uninstall-agent.sh | sudo bash

# Or directly from the GitHub repository:
curl -sSL https://raw.githubusercontent.com/depo-platform/depo/main/scripts/uninstall-agent.sh | sudo bash
```
To also purge all deployed static website releases located in `/var/www`:
```bash
curl -sSL https://depo.example.com/uninstall-agent.sh | sudo bash -s -- --purge-www
```

---

## CLI Options

### `depo` (Central Control Plane)
```text
Usage of depo:
  --port int
        HTTP port for the central dashboard & API (default 8080)
  --data string
        Filesystem directory for SQLite DB, encryption keys, and artifacts (default "./data")
  --url string
        Public URL of the DEPO control plane (default "http://localhost:8080")
```

### `depo-agent` (VPS Edge Worker)
```text
Usage of depo-agent:
  enroll
        Enroll this server with a central control plane
        Flags: --server <URL> --token <ENROLL_TOKEN> --config <PATH>
  run
        Run the edge agent worker daemon
        Flags: --config <PATH> (default "/etc/depo-agent/config.json")
```

---

## Verification & Testing

DEPO includes an extensive automated test suite covering database transactions, job lease safety, zip-slip defense, rate limiting, and Caddyfile generation:

```bash
# Run all tests
go test -v -count=1 ./...

# Build frontend verification
cd web
npm run build
```

---

## Contributing

Contributions are welcome! Please review [CONTRIBUTING.md](CONTRIBUTING.md) for details on code standards, local development workflows, and pull request submissions.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
