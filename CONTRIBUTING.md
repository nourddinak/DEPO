# Contributing to DEPO

Thank you for your interest in contributing to DEPO! This guide covers how to set up your development environment, run tests, and submit changes.

---

## Architecture Overview

DEPO is structured into three primary subsystems:

1. **Central Control Plane (`cmd/depo`)**:
   - Written in Go.
   - Manages projects, multi-tenant users, deployment manifests, artifacts, and server agent heartbeats.
   - Embeds the compiled React SPA directly using `embed.FS` for single-binary distribution.
   - Built on pure-Go SQLite (`modernc.org/sqlite`) with WAL mode and zero CGO dependencies.
   - Stored in `internal/api`, `internal/db`, `internal/crypto`.

2. **Edge Deployment Agent (`cmd/depo-agent`)**:
   - Lightweight, standalone Go daemon running on each target VPS.
   - Long-polls central control plane for deployment tasks via authenticated TLS endpoints.
   - Safely extracts tar.gz web artifacts (with zip-slip protections), verifies SHA-256 hashes, atomically swaps symlinks (`current -> releases/<id>`), and programmatically reloads Caddy via Caddy's dynamic JSON admin API.
   - Stored in `internal/agent`.

3. **Dashboard Web UI (`web/`)**:
   - Modern React 19 + TypeScript + Vite + Tailwind CSS + Lucide icons.
   - Compiled assets are mirrored into `cmd/depo/dist` for binary embedding.

---

## Local Development Setup

### Prerequisites
- **Go**: 1.24 or later
- **Node.js**: 22.x LTS or later (with `npm`)
- **Git**

### 1. Clone Repository
```bash
git clone https://github.com/depo-platform/depo.git
cd depo
```

### 2. Run Backend in Development
```bash
go run ./cmd/depo --data ./data --port 8080
```
The central API will listen on `http://localhost:8080`.

### 3. Run Frontend with Hot Module Replacement (HMR)
In a separate terminal window:
```bash
cd web
npm install
npm run dev
```
Vite will start the frontend development server on `http://localhost:5173`, automatically proxying `/api` requests to `http://localhost:8080`.

---

## Building & Testing

### Running Backend Tests
```bash
go test -v -count=1 ./...
```

### Typechecking & Building Frontend
```bash
cd web
npm run build
```

### Syncing Frontend to Embedded Go Assets
When frontend modifications are ready to be included in the single binary build:
```bash
# From repository root
cd web
npm run build
cd ..
# In PowerShell:
Copy-Item -Recurse -Force web\dist\* cmd\depo\dist\
# In Bash:
# cp -r web/dist/* cmd/depo/dist/
```

### Compiling Standalone Binaries
```bash
# Control plane server
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/depo ./cmd/depo

# Edge agent daemon
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/depo-agent ./cmd/depo-agent
```

---

## Pull Request Guidelines

1. **Keep Changes Focused**: Address a single feature or bug per pull request. Avoid drive-by reformatting or touching unrelated packages.
2. **Add Tests**: Provide automated unit tests for any new API endpoints, database methods, or agent execution routines.
3. **Run Full Verification**:
   - `go test -count=1 ./...` must exit 0.
   - `cd web && npm run build` must exit 0 with zero TypeScript errors.
4. **Security & Secrets**: Never commit real database files (`data/depo.db*`), cryptographic keys (`data/master.key`), or production secrets.

---

## License

By contributing to DEPO, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
