# ==============================================================================
# DEPO Multi-Stage Production Container Build
# Self-contained: builds frontend SPA + compiles zero-dependency Go binaries
# ==============================================================================

# Stage 1: Build React + Vite Dashboard
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Compile Central Server & Agent Binaries
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY scripts/ ./scripts/
# Embed fresh compiled frontend into cmd/depo/dist
COPY --from=web-builder /app/web/dist ./cmd/depo/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/depo ./cmd/depo
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/depo-agent ./cmd/depo-agent

# Stage 3: Minimal Secure Production Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata bash curl

WORKDIR /var/lib/depo
RUN mkdir -p /var/lib/depo/artifacts

COPY --from=go-builder /app/bin/depo /usr/local/bin/depo
COPY --from=go-builder /app/bin/depo-agent /usr/local/bin/depo-agent
COPY scripts/ /usr/local/share/depo/scripts/

EXPOSE 8080
VOLUME ["/var/lib/depo"]

ENTRYPOINT ["/usr/local/bin/depo"]
CMD ["--data", "/var/lib/depo", "--port", "8080"]
