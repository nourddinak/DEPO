package main

import (
	"context"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"depo/internal/api"
	"depo/internal/crypto"
	"depo/internal/db"
)

//go:embed all:dist
var embeddedFS embed.FS

var Version = "1.0.0"

func main() {
	portFlag := flag.Int("port", 0, "HTTP port to listen on (default 8080 or PORT env var)")
	dataDirFlag := flag.String("data", "", "Directory for SQLite database and artifacts (default ./data or DEPO_DATA_DIR env var)")
	secretFlag := flag.String("secret", "", "AES-256 master encryption key (default DEPO_SECRET_KEY env var or auto-generated in data/master.key)")
	urlFlag := flag.String("url", "", "External platform URL (default http://localhost:8080 or DEPO_EXTERNAL_URL env var)")
	webhookSecretFlag := flag.String("webhook-secret", "", "GitHub webhook HMAC secret (default DEPO_WEBHOOK_SECRET env var)")
	flag.Parse()

	// Resolve Port
	port := 8080
	if *portFlag > 0 {
		port = *portFlag
	} else if pStr := os.Getenv("PORT"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			port = p
		}
	}

	// Resolve Data Directory
	dataDir := "./data"
	if *dataDirFlag != "" {
		dataDir = *dataDirFlag
	} else if d := os.Getenv("DEPO_DATA_DIR"); d != "" {
		dataDir = d
	}
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		log.Fatalf("Failed to initialize data directory %s: %v", dataDir, err)
	}

	// Resolve Master Encryption Key (Hybrid approach)
	masterKey, err := resolveMasterKey(*secretFlag, dataDir)
	if err != nil {
		log.Fatalf("Master key initialization error: %v", err)
	}

	// Resolve External URL
	externalURL := "http://localhost:8080"
	if *urlFlag != "" {
		externalURL = *urlFlag
	} else if u := os.Getenv("DEPO_EXTERNAL_URL"); u != "" {
		externalURL = u
	}

	// Resolve Webhook Secret
	webhookSecret := os.Getenv("DEPO_WEBHOOK_SECRET")
	if *webhookSecretFlag != "" {
		webhookSecret = *webhookSecretFlag
	}

	// Initialize Database (SQLite WAL + Migrations)
	dbPath := filepath.Join(dataDir, "depo.db")
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database at %s: %v", dbPath, err)
	}
	defer database.Close()

	// Extract web SPA static assets
	webSubFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		log.Printf("[warning] Could not load embedded web filesystem: %v", err)
	}

	// Initialize Central API Server
	server := api.NewServer(api.Config{
		DB:            database,
		MasterKey:     masterKey,
		DataDir:       dataDir,
		ExternalURL:   externalURL,
		WebhookSecret: webhookSecret,
		WebFS:         webSubFS,
	})

	// Start Background Maintenance Worker (offline detection, session expiry)
	stopWorkerCh := make(chan struct{})
	go server.StartBackgroundWorker(stopWorkerCh)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      server.Routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("=============================================================")
		log.Printf(" DEPO Multi-VPS Static Frontend Platform v%s", Version)
		log.Printf(" Listening on: %s (port %d)", externalURL, port)
		log.Printf(" SQLite DB:    %s", dbPath)
		log.Printf(" Artifacts:    %s/artifacts", dataDir)
		log.Printf("=============================================================")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("\n[server] Shutting down gracefully...")
	close(stopWorkerCh)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[server] Forced shutdown error: %v", err)
	}

	log.Println("[server] Exited cleanly.")
}

func resolveMasterKey(flagVal, dataDir string) ([]byte, error) {
	// 1. Check flag or environment variable
	keyHex := flagVal
	if keyHex == "" {
		keyHex = os.Getenv("DEPO_SECRET_KEY")
	}

	if keyHex != "" {
		key, err := hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("decoding hex DEPO_SECRET_KEY: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("DEPO_SECRET_KEY must be exactly 32 bytes (64 hex characters), got %d bytes", len(key))
		}
		return key, nil
	}

	// 2. Check data/master.key file
	keyPath := filepath.Join(dataDir, "master.key")
	if keyData, err := os.ReadFile(keyPath); err == nil && len(keyData) == 32 {
		return keyData, nil
	}

	// 3. Generate new 256-bit key and save to data/master.key with 0600 permissions
	newKey, err := crypto.GenerateRandomKey()
	if err != nil {
		return nil, fmt.Errorf("generating master key: %w", err)
	}

	if err := os.WriteFile(keyPath, newKey, 0600); err != nil {
		return nil, fmt.Errorf("persisting master key to %s: %w", keyPath, err)
	}

	log.Printf("[security] Generated new AES-256 master encryption key in %s", keyPath)
	return newKey, nil
}
