package api

import (
	"io/fs"
	"net/http"
	"strings"
	"time"

	"depo/internal/db"
	"depo/internal/github"
	"depo/internal/notify"
)

type Server struct {
	db            *db.DB
	masterKey     []byte
	dataDir       string
	externalURL   string
	webhookSecret string
	sseHub        *SSEHub
	rateLimiter   *RateLimiter
	notifier      *notify.Dispatcher
	oidcValidator *github.OIDCValidator
	webFS         fs.FS
}

type Config struct {
	DB            *db.DB
	MasterKey     []byte
	DataDir       string
	ExternalURL   string
	WebhookSecret string
	WebFS         fs.FS
}

func NewServer(cfg Config) *Server {
	if cfg.ExternalURL == "" {
		cfg.ExternalURL = "http://localhost:8080"
	}
	return &Server{
		db:            cfg.DB,
		masterKey:     cfg.MasterKey,
		dataDir:       cfg.DataDir,
		externalURL:   cfg.ExternalURL,
		webhookSecret: cfg.WebhookSecret,
		sseHub:        NewSSEHub(),
		rateLimiter:   NewRateLimiter(15, 1*time.Minute), // 15 attempts/min for login
		notifier:      notify.NewDispatcher(),
		oidcValidator: github.NewOIDCValidator("depo-deploy"),
		webFS:         cfg.WebFS,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// --- Public Auth Endpoints ---
	mux.HandleFunc("GET /api/v1/auth/status", s.handleAuthStatus)
	mux.HandleFunc("POST /api/v1/auth/setup", s.handleSetup)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/invitations/accept", s.handleAcceptInvitation)

	// --- Public Script & Agent Download Endpoints ---
	mux.HandleFunc("GET /install-agent.sh", s.handleInstallAgentScript)
	mux.HandleFunc("GET /install.sh", s.handleInstallServerScript)
	mux.HandleFunc("GET /downloads/{name}", s.handleAgentDownload)

	// --- Webhooks & Artifacts ---
	mux.HandleFunc("POST /api/v1/webhooks/github", s.handleGitHubWebhook)
	mux.HandleFunc("POST /api/v1/artifacts/upload", s.handleArtifactUpload)
	mux.HandleFunc("GET /api/v1/artifacts/{id}/download", s.handleArtifactDownload)

	// --- Agent-Facing Endpoints ---
	mux.HandleFunc("POST /api/v1/agent/enroll", s.handleAgentEnroll)
	mux.HandleFunc("POST /api/v1/agent/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("GET /api/v1/agent/jobs/poll", s.handleAgentPollJobs)
	mux.HandleFunc("POST /api/v1/agent/jobs/{id}/progress", s.handleAgentJobProgress)
	mux.HandleFunc("POST /api/v1/agent/jobs/{id}/complete", s.handleAgentJobComplete)
	mux.HandleFunc("POST /api/v1/agent/jobs/{id}/fail", s.handleAgentJobFail)

	// --- SSE Realtime Stream ---
	mux.Handle("/api/v1/events", s.sseHub)

	// --- Protected User / Dashboard Endpoints ---
	mux.HandleFunc("GET /api/v1/auth/me", RequireAuth(s.handleMe))
	mux.HandleFunc("POST /api/v1/auth/totp/setup", RequireAuth(s.handleTOTPSetup))
	mux.HandleFunc("POST /api/v1/auth/totp/verify", RequireAuth(s.handleTOTPVerify))
	mux.HandleFunc("GET /api/v1/auth/invitations", RequireAuth(s.handleListInvitations))
	mux.HandleFunc("POST /api/v1/auth/invitations", RequireAuth(s.handleCreateInvitation))
	mux.HandleFunc("GET /api/v1/users", RequireAuth(s.handleListUsers))
	mux.HandleFunc("GET /api/v1/audit-logs", RequireAuth(s.handleListAuditLogs))

	// Servers
	mux.HandleFunc("GET /api/v1/servers", RequireAuth(s.handleListServers))
	mux.HandleFunc("POST /api/v1/servers", RequireAuth(s.handleCreateServer))
	mux.HandleFunc("GET /api/v1/servers/{id}", RequireAuth(s.handleGetServer))
	mux.HandleFunc("DELETE /api/v1/servers/{id}", RequireAuth(s.handleDeleteServer))

	// Projects
	mux.HandleFunc("POST /api/v1/projects/detect", RequireAuth(s.handleDetectFramework))
	mux.HandleFunc("GET /api/v1/projects", RequireAuth(s.handleListProjects))
	mux.HandleFunc("POST /api/v1/projects", RequireAuth(s.handleCreateProject))
	mux.HandleFunc("GET /api/v1/projects/{id}", RequireAuth(s.handleGetProject))
	mux.HandleFunc("PUT /api/v1/projects/{id}", RequireAuth(s.handleUpdateProject))
	mux.HandleFunc("DELETE /api/v1/projects/{id}", RequireAuth(s.handleDeleteProject))
	mux.HandleFunc("GET /api/v1/projects/{id}/workflow", RequireAuth(s.handleGetProjectWorkflow))

	// Deployments & Releases
	mux.HandleFunc("GET /api/v1/deployments", RequireAuth(s.handleListDeployments))
	mux.HandleFunc("GET /api/v1/deployments/{id}", RequireAuth(s.handleGetDeployment))
	mux.HandleFunc("GET /api/v1/deployments/{id}/logs", RequireAuth(s.handleGetDeploymentLogs))
	mux.HandleFunc("GET /api/v1/projects/{id}/releases", RequireAuth(s.handleListReleases))
	mux.HandleFunc("POST /api/v1/projects/{id}/rollback", RequireAuth(s.handleRollbackProject))

	// Previews
	mux.HandleFunc("GET /api/v1/previews", RequireAuth(s.handleListPreviews))
	mux.HandleFunc("GET /api/v1/previews/{id}", RequireAuth(s.handleGetPreview))
	mux.HandleFunc("POST /api/v1/previews/{id}/destroy", RequireAuth(s.handleDestroyPreview))

	// Environment Variables
	mux.HandleFunc("GET /api/v1/projects/{id}/env", RequireAuth(s.handleGetProjectEnv))
	mux.HandleFunc("POST /api/v1/projects/{id}/env", RequireAuth(s.handleUpdateProjectEnv))
	mux.HandleFunc("POST /api/v1/projects/parse-env-example", RequireAuth(s.handleParseEnvExample))

	// Notifications & Diagnostics
	mux.HandleFunc("GET /api/v1/settings/notifications", RequireAuth(s.handleGetNotificationSettings))
	mux.HandleFunc("PUT /api/v1/settings/notifications", RequireAuth(s.handleUpdateNotificationSettings))
	mux.HandleFunc("POST /api/v1/settings/notifications/test", RequireAuth(s.handleTestNotification))
	mux.HandleFunc("GET /api/v1/domains/check", RequireAuth(s.handleCheckDomain))

	// --- Static Web Assets & Single Page Application (SPA) Fallback ---
	if s.webFS != nil {
		fileServer := http.FileServer(http.FS(s.webFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Don't intercept API routes
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			// Try to open file directly
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			if f, err := s.webFS.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA Fallback: serve index.html for client-side routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	// Apply Middlewares in order: RequestID -> SecurityHeaders -> Auth
	var handler http.Handler = mux
	handler = s.AuthMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = RequestIDMiddleware(handler)

	return handler
}

// BackgroundWorker monitors server offline timeouts and expired previews
func (s *Server) StartBackgroundWorker(stopCh <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			// 1. Mark servers with no heartbeat for > 60s as offline
			offlineIDs, err := s.db.MarkInactiveServersOffline(60 * time.Second)
			if err == nil && len(offlineIDs) > 0 {
				for _, sid := range offlineIDs {
					s.sseHub.Broadcast("server_status", map[string]any{
						"server_id": sid,
						"status":    "offline",
					})
				}
			}

			// 2. Clean expired sessions
			_ = s.db.DeleteExpiredSessions()
		}
	}
}
